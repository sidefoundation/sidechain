package keeper_test

import (
	"fmt"
	"sidechain/x/devearn/types"
)

func (suite *KeeperTestSuite) TestGetAllAssets() {
	var expRes []types.Assets

	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			"no asset registered",
			func() { expRes = []types.Assets{} },
		},
		{
			"1 pair registered",
			func() {
				in := types.Assets{Denom: "coin"}
				suite.app.DevearnKeeper.SetAssets(suite.ctx, in)
				suite.Commit()

				expRes = []types.Assets{in}
			},
		},
		{
			"2 pairs registered",
			func() {
				in := types.Assets{Denom: "coin"}
				in2 := types.Assets{Denom: "coin2"}
				suite.app.DevearnKeeper.SetAssets(suite.ctx, in)
				suite.app.DevearnKeeper.SetAssets(suite.ctx, in2)
				suite.Commit()

				expRes = []types.Assets{in, in2}
			},
		},
		{
			"6 pairs registered",
			func() {
				in := types.Assets{Denom: "aside"}
				in2 := types.Assets{Denom: "bside"}
				in3 := types.Assets{Denom: "IBC/C4CFF46FD6DE35CA4CF4CE031E643C8FDC9BA4B99AE598E9B0ED98FE3A2319F9"}
				in4 := types.Assets{Denom: "IBC/C4CFF46FD6DE35CA4CF4CE031E643C8FASD9BA4B99AE59E9B0ED98FE3A2319F9"}
				in5 := types.Assets{Denom: "IBC/C4CFF46FD6DE35DFGCF4CE031E643C8FDC9BA4B99AE598E9B0ED98FE3A2319F9"}
				in6 := types.Assets{Denom: "IBC/C4CFF46FD6DE3GHJ4CF4CE031E643C8FDC9BA4B99AE598E9B0ED98FE3A2319F9"}

				suite.app.DevearnKeeper.SetAssets(suite.ctx, in)
				suite.app.DevearnKeeper.SetAssets(suite.ctx, in2)
				suite.app.DevearnKeeper.SetAssets(suite.ctx, in3)
				suite.app.DevearnKeeper.SetAssets(suite.ctx, in4)
				suite.app.DevearnKeeper.SetAssets(suite.ctx, in5)
				suite.app.DevearnKeeper.SetAssets(suite.ctx, in6)

				suite.Commit()

				expRes = []types.Assets{in, in2, in3, in4, in5, in6}
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset
			tc.malleate()
			res := suite.app.DevearnKeeper.GetAllAssets(suite.ctx)
			suite.Require().ElementsMatch(expRes, res, tc.name)
		})
	}
}

func (suite *KeeperTestSuite) TestDeleteAssets() {
	// Add asset
	suite.app.DevearnKeeper.SetAssets(
		suite.ctx,
		types.Assets{Denom: "coin"},
	)

	found := suite.app.DevearnKeeper.IsAssetRegistered(suite.ctx, "coin")
	suite.Require().True(found)

	testCases := []struct {
		name     string
		malleate func()
		ok       bool
	}{
		{"valid asset", func() {}, true},
		{
			"deleted asset",
			func() {
				suite.app.DevearnKeeper.RemoveAssetFromWhitelist(suite.ctx, "coin")
			},
			false,
		},
	}
	for _, tc := range testCases {
		tc.malleate()
		found := suite.app.DevearnKeeper.IsAssetRegistered(
			suite.ctx,
			"coin",
		)
		if tc.ok {
			suite.Require().True(found, tc.name)
		} else {
			suite.Require().False(found, tc.name)
		}
	}
}

func (suite *KeeperTestSuite) TestIsAssetRegistered() {
	suite.SetupTest() // reset
	suite.app.DevearnKeeper.SetAssets(suite.ctx, types.Assets{Denom: "coin"})
	suite.Commit()

	testCases := []struct {
		name  string
		denom string
		ok    bool
	}{
		{"valid id", "coin", true},
		{"pair not found", "coin2", false},
	}
	for _, tc := range testCases {
		found := suite.app.DevearnKeeper.IsAssetRegistered(
			suite.ctx,
			tc.denom,
		)
		if tc.ok {
			suite.Require().True(found, tc.name)
		} else {
			suite.Require().False(found, tc.name)
		}
	}
}
