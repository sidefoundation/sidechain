package keeper

import (
	"math/big"
	"sidechain/contracts"
	"sidechain/x/devearn/types"
	"strconv"

	erc20types "sidechain/x/erc20/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// DistributeRewards transfers the allocated rewards to the participants
//   - allocates the amount to be distributed from the inflation pool
//   - distributes the rewards to all participants
//   - deletes all gas meters
//   - updates the remaining epochs of each incentive
//   - sets the cumulative totalGas to zero
func (k Keeper) DistributeRewards(ctx sdk.Context) error {
	logger := k.Logger(ctx)
	devEarnGasMeters := make(map[string]uint64)
	devEarnRewardReceivers := make(map[string]string)
	k.IterateDevEarnInfos(ctx, func(devEarnInfo types.DevEarnInfo) (stop bool) {
		devEarnGasMeters[devEarnInfo.GetContract()] = devEarnInfo.GetGasMeter()
		devEarnRewardReceivers[devEarnInfo.GetContract()] = devEarnInfo.GetOwnerAddress()
		devEarnInfo.Epochs--

		// Update dev_earn info and reset its total gas count. Remove dev_earn info if it
		// has no remaining epochs left.
		if devEarnInfo.IsActive() {
			devEarnInfo.GasMeter = 0
			k.SetDevEarnInfo(ctx, devEarnInfo)
		} else {
			k.DeleteDevEarnInfo(ctx, devEarnInfo)
			logger.Info(
				"devEarn finalized",
				"contract", devEarnInfo.Contract,
			)
		}

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeDistributeRewards,
				sdk.NewAttribute(types.AttributeKeyContract, devEarnInfo.Contract),
				sdk.NewAttribute(
					types.AttributeKeyEpochs,
					strconv.FormatUint(uint64(devEarnInfo.Epochs), 10),
				),
			),
		)
		return false
	})
	k.SendReward(ctx, devEarnGasMeters, devEarnRewardReceivers)

	return nil
}

func (k Keeper) SendReward(
	ctx sdk.Context,
	devEarnGasMeters map[string]uint64,
	devEarnRewardReceivers map[string]string,
) (rewards sdk.Coins, count uint64) {
	logger := k.Logger(ctx)
	var totalGasDec sdk.Dec = sdk.NewDec(0)
	// Check if participants spent gas on interacting with incentive
	for _, gasMeter := range devEarnGasMeters {
		totalGasDec = totalGasDec.Add(sdk.NewDecFromBigInt(new(big.Int).SetUint64(gasMeter)))
	}
	if totalGasDec.Equal(sdk.NewDec(0)) {
		logger.Debug(
			"no gas spent on dev earn during epoch",
		)
		return sdk.Coins{}, 0
	}
	moduleAddr := k.accountKeeper.GetModuleAddress(types.ModuleName)
	// TODO: Check base denom
	denom := "aside"
	// denom, err := sdk.GetBaseDenom()
	// if err != nil {
	// 	logger.Debug("could not get the denom of smallest unit registered", "error", err.Error())
	// }
	totalReward := k.bankKeeper.GetBalance(ctx, moduleAddr, denom)
	// Get fees, tvl split
	split := k.GetParams(ctx).TvlShare
	for contract, gasmeter := range devEarnGasMeters {
		cumulativeGas := sdk.NewDecFromBigInt(new(big.Int).SetUint64(gasmeter))
		gasRatio := cumulativeGas.Quo(totalGasDec)
		tvlRatio, tvlErr := k.TvlReward(ctx, contract)
		if tvlErr != nil {
			logger.Debug("could not get tvl ratio", "error", tvlErr.Error())
		}

		// split total reward using tvl_param in parameters
		rewardTvlSplit := sdk.NewDecFromInt(totalReward.Amount).Mul(sdk.NewDecFromBigInt(new(big.Int).SetUint64(split)))
		rewardTvlSplit = rewardTvlSplit.QuoInt(sdk.NewInt(10000))
		rewardGasSplit := sdk.NewDecFromInt(totalReward.Amount).Sub(rewardTvlSplit)
		reward := (gasRatio.Mul(rewardGasSplit)).Add(tvlRatio.Mul(rewardTvlSplit))

		if !reward.IsPositive() {
			continue
		}

		coin := sdk.Coin{Denom: denom, Amount: reward.TruncateInt()}
		coins := sdk.Coins{}
		coins = coins.Add(coin)

		participant := common.HexToAddress(devEarnRewardReceivers[contract])
		err := k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx,
			types.ModuleName,
			sdk.AccAddress(participant.Bytes()),
			coins,
		)
		if err != nil {
			logger.Debug("failed to distribute developer's rewards",
				"address", devEarnRewardReceivers[contract],
				"allocation", coins.String(),
				"contract_addr", contract,
				"error", err.Error(),
			)
		}
	}
	return rewards, count
}

// TvlReward function calculates TVL rewards using assets in whitelist
func (k Keeper) TvlReward(ctx sdk.Context, contractAddress string) (sdk.Dec, error) {
	assets := k.GetAllAssets(ctx)
	totalValueLocked, tvlErr := k.TotalTvl(ctx)
	if tvlErr != nil {
		return sdk.NewDec(0), tvlErr
	}
	totalValueLockedContract := sdk.NewDec(0)
	// What should happen if one of the values is not loaded ? return 0 or cancel the process
	for i := 0; i < len(assets); i++ {
		// Get exchange rate using oracle module
		rate, err := k.oracleKeeper.GetExchangeRate(ctx, assets[i].Denom)
		if err != nil {
			return sdk.NewDec(0), err
		}

		// Get mapping to erc20 token from cosmos denom
		tokenPair, tokenPairErr := k.erc20Keeper.TokenPair(
			ctx, &erc20types.QueryTokenPairRequest{Token: assets[i].Denom})
		if tokenPairErr != nil {
			return sdk.NewDec(0), tokenPairErr
		}
		erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI

		// Get balance from erc20 token
		tokenBalance := k.erc20Keeper.BalanceOf(
			ctx, erc20, tokenPair.GetTokenPair().GetERC20Contract(), common.HexToAddress(contractAddress))

		totalValueLockedContract = totalValueLockedContract.Add(sdk.NewDecFromBigInt(tokenBalance).Mul(rate))
	}

	if totalValueLocked.IsZero() {
		return sdk.NewDec(0), nil
	}
	tvlRatio := totalValueLockedContract.Quo(totalValueLocked)

	return tvlRatio, nil
}

func (k Keeper) TotalTvl(ctx sdk.Context) (sdk.Dec, error) {
	assets := k.GetAllAssets(ctx)
	totalValueLocked := sdk.NewDec(0)
	// What should happen if one of the values is not loaded ? return 0 or cancel the process
	for i := 0; i < len(assets); i++ {
		// Get exchange rate using oracle module
		rate, err := k.oracleKeeper.GetExchangeRate(ctx, assets[i].Denom)
		if err != nil {
			return sdk.NewDec(0), err
		}
		erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
		// Get mapping to erc20 token from cosmos denom
		tokenPair, tokenPairErr := k.erc20Keeper.TokenPair(
			ctx, &erc20types.QueryTokenPairRequest{Token: assets[i].Denom})
		if tokenPairErr != nil {
			return sdk.NewDec(0), tokenPairErr
		}
		total := k.erc20Keeper.TotalSupply(ctx, erc20, tokenPair.TokenPair.GetERC20Contract())
		totalValueLocked = totalValueLocked.Add(sdk.NewDecFromBigInt(total).Mul(rate))
	}
	return totalValueLocked, nil
}
