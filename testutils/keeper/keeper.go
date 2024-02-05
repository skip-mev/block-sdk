// Package keeper provides methods to initialize SDK keepers with local storage for test purposes
package keeper

import (
	"testing"

	"cosmossdk.io/math"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	testkeeper "github.com/skip-mev/chaintestutil/keeper"
	"github.com/stretchr/testify/require"

	"github.com/skip-mev/block-sdk/v2/lanes/base"
	"github.com/skip-mev/block-sdk/v2/lanes/free"
	"github.com/skip-mev/block-sdk/v2/lanes/mev"
	auctionkeeper "github.com/skip-mev/block-sdk/v2/x/auction/keeper"
	auctiontypes "github.com/skip-mev/block-sdk/v2/x/auction/types"
	blocksdkkeeper "github.com/skip-mev/block-sdk/v2/x/blocksdk/keeper"
	blocksdktypes "github.com/skip-mev/block-sdk/v2/x/blocksdk/types"
)

// TestKeepers holds all keepers used during keeper tests for all modules
type TestKeepers struct {
	testkeeper.TestKeepers
	AuctionKeeper  auctionkeeper.Keeper
	BlockSDKKeeper blocksdkkeeper.Keeper
}

// TestMsgServers holds all message servers used during keeper tests for all modules
type TestMsgServers struct {
	testkeeper.TestMsgServers
	AuctionMsgServer auctiontypes.MsgServer
}

var additionalMaccPerms = map[string][]string{
	auctiontypes.ModuleName: nil,
}

// NewTestSetup returns initialized instances of all the keepers and message servers of the modules
func NewTestSetup(t testing.TB, options ...testkeeper.SetupOption) (sdk.Context, TestKeepers, TestMsgServers) {
	options = append(options, testkeeper.WithAdditionalModuleAccounts(additionalMaccPerms))

	_, tk, tms := testkeeper.NewTestSetup(t, options...)

	// initialize extra keeper
	auctionKeeper := Auction(tk.Initializer, tk.AccountKeeper, tk.BankKeeper, tk.DistrKeeper, tk.StakingKeeper)
	blocksdkKeeper := BlockSDK(tk.Initializer)
	require.NoError(t, tk.Initializer.LoadLatest())

	// initialize msg servers
	auctionMsgSrv := auctionkeeper.NewMsgServerImpl(auctionKeeper)

	ctx := sdk.NewContext(tk.Initializer.StateStore, tmproto.Header{
		Time:   testkeeper.ExampleTimestamp,
		Height: testkeeper.ExampleHeight,
	}, false, log.NewNopLogger())

	err := auctionKeeper.SetParams(ctx, auctiontypes.DefaultParams())
	require.NoError(t, err)

	err = blocksdkKeeper.AddLane(ctx, blocksdktypes.Lane{
		Id:            mev.LaneName,
		MaxBlockSpace: math.LegacyMustNewDecFromStr("0.2"),
		Order:         0,
	})
	require.NoError(t, err)
	err = blocksdkKeeper.AddLane(ctx, blocksdktypes.Lane{
		Id:            free.LaneName,
		MaxBlockSpace: math.LegacyMustNewDecFromStr("0.2"),
		Order:         1,
	})
	require.NoError(t, err)

	err = blocksdkKeeper.AddLane(ctx, blocksdktypes.Lane{
		Id:            base.LaneName,
		MaxBlockSpace: math.LegacyMustNewDecFromStr("0.6"),
		Order:         2,
	})
	require.NoError(t, err)

	testKeepers := TestKeepers{
		TestKeepers:    tk,
		AuctionKeeper:  auctionKeeper,
		BlockSDKKeeper: blocksdkKeeper,
	}

	testMsgServers := TestMsgServers{
		TestMsgServers:   tms,
		AuctionMsgServer: auctionMsgSrv,
	}

	return ctx, testKeepers, testMsgServers
}

// Auction initializes the auction module using the testkeepers intializer.
func Auction(
	initializer *testkeeper.Initializer,
	authKeeper authkeeper.AccountKeeper,
	bankKeeper bankkeeper.Keeper,
	distrKeeper distrkeeper.Keeper,
	stakingKeeper *stakingkeeper.Keeper,
) auctionkeeper.Keeper {
	storeKey := storetypes.NewKVStoreKey(auctiontypes.StoreKey)
	initializer.StateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, initializer.DB)

	return auctionkeeper.NewKeeper(
		initializer.Codec,
		storeKey,
		authKeeper,
		bankKeeper,
		distrKeeper,
		stakingKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
}

// BlockSDK initializes the blocksdk module using the testkeepers intializer.
func BlockSDK(
	initializer *testkeeper.Initializer,
) blocksdkkeeper.Keeper {
	storeKey := storetypes.NewKVStoreKey(blocksdktypes.StoreKey)
	initializer.StateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, initializer.DB)

	return blocksdkkeeper.NewKeeper(
		initializer.Codec,
		storeKey,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
}
