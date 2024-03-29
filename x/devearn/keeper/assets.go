package keeper

import (
	"sidechain/x/devearn/types"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetAssets set a specific assets in the store
func (k Keeper) SetAssets(ctx sdk.Context, assets types.Assets) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AssetsKey))
	b := k.cdc.MustMarshal(&assets)
	store.Set(GetAssetBytes(assets.Denom), b)
}

// IsAssetRegistered - check if asset is in whitelist
func (k Keeper) IsAssetRegistered(
	ctx sdk.Context,
	denom string,
) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AssetsKey))
	return store.Has(GetAssetBytes(denom))
}

// RemoveAssets removes a assets from the store
func (k Keeper) RemoveAssets(ctx sdk.Context, denom string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AssetsKey))
	store.Delete(GetAssetBytes(denom))
}

// GetAllAssets returns all assets
func (k Keeper) GetAllAssets(ctx sdk.Context) (list []types.Assets) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AssetsKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Assets
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetAssetsBytes returns the byte representation of the denom
func GetAssetBytes(denom string) []byte {
	return []byte(denom)
}

// GetAssetsFromBytes returns denom in string format from a byte array
// func GetAssetFromBytes(bz []byte) string {
// 	return binary.BigEndian.Uint64(bz)
// }
