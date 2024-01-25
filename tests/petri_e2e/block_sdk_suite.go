package petri_e2e

import (
	"context"
	_ "embed"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	comettypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/informalsystems/tm-load-test/pkg/loadtest"
	"github.com/skip-mev/block-sdk/tests/app"
	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"
	blocksdkmoduletypes "github.com/skip-mev/block-sdk/x/auction/types"
	"github.com/skip-mev/petri/chain"
	"github.com/skip-mev/petri/cosmosutil"
	petriload "github.com/skip-mev/petri/loadtest"
	"github.com/skip-mev/petri/monitoring"
	"github.com/skip-mev/petri/provider"
	"github.com/skip-mev/petri/provider/docker"
	petritypes "github.com/skip-mev/petri/types"
	"github.com/skip-mev/petri/wallet"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

//go:embed files/dashboard.json
var dashboardJSON string

// E2ETestSuite runs the Block SDK e2e test-suite against a given interchaintest specification
type E2ETestSuite struct {
	suite.Suite

	// spec
	spec petritypes.ChainConfig

	chain       petritypes.ChainI
	chainClient *cosmosutil.ChainClient

	// provider
	provider provider.Provider

	// monitoring
	grafanaTask *provider.Task

	// users
	user1, user2, user3 *cosmosutil.InteractingWallet
	// denom
	denom string
	// fuzzusers
	fuzzusers []*cosmosutil.InteractingWallet
}

func NewE2ETestSuiteFromSpec(spec petritypes.ChainConfig) *E2ETestSuite {
	return &E2ETestSuite{
		spec:  spec,
		denom: "stake",
	}
}

func AMakeEncodingConfig() *testutil.TestEncodingConfig {
	cfg := testutil.MakeTestEncodingConfig()
	authtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	banktypes.RegisterInterfaces(cfg.InterfaceRegistry)

	// register auction types
	auctiontypes.RegisterInterfaces(cfg.InterfaceRegistry)
	blocksdkmoduletypes.RegisterInterfaces(cfg.InterfaceRegistry)

	return &cfg
}

func (s *E2ETestSuite) SetupSuite() {
	sdk.GetConfig().SetBech32PrefixForAccount(s.spec.Bech32Prefix, s.spec.Bech32Prefix)
	logger, err := zap.NewDevelopment()
	s.Require().NoError(err)

	ctx := context.Background()

	encodingConfig := AMakeEncodingConfig()

	provider, err := docker.NewDockerProvider(ctx, logger, "petri_docker")
	s.Require().NoError(err)
	s.provider = provider

	// create the chain
	chain, err := chain.CreateChain(ctx, logger, provider, s.spec)
	s.Require().NoError(err)

	s.chain = chain
	fmt.Println(s.chain.GetConfig().Denom)

	var endpoints []string

	for _, node := range append(s.chain.GetValidators(), s.chain.GetNodes()...) {
		endpoint, err := node.GetTask().GetIP(ctx)
		s.Require().NoError(err)

		endpoints = append(endpoints, fmt.Sprintf("%s:26660", endpoint))
	}

	prometheusTask, err := monitoring.SetupPrometheusTask(ctx, logger, provider, monitoring.PrometheusOptions{
		Targets: endpoints,
	})
	s.Require().NoError(err)

	err = prometheusTask.Start(ctx, false)
	s.Require().NoError(err)

	prometheusIP, err := prometheusTask.GetExternalAddress(ctx, "9090/tcp")
	s.Require().NoError(err)

	grafanaTask, err := monitoring.SetupGrafanaTask(ctx, logger, provider, monitoring.GrafanaOptions{
		DashboardJSON: dashboardJSON,
		PrometheusURL: fmt.Sprintf("http://%s:9090", prometheusIP),
	})
	s.Require().NoError(err)

	s.grafanaTask = grafanaTask

	err = grafanaTask.Start(ctx, false)
	s.Require().NoError(err)

	grafanaIP, err := s.grafanaTask.GetExternalAddress(ctx, "3000/tcp")

	fmt.Printf("Visit Grafana at http://%s\n", grafanaIP)

	s.chainClient = &cosmosutil.ChainClient{Chain: chain, EncodingConfig: cosmosutil.EncodingConfig{
		InterfaceRegistry: encodingConfig.InterfaceRegistry,
		TxConfig:          encodingConfig.TxConfig,
		Codec:             encodingConfig.Codec,
	}}

	// create the users
	wallet1, err := wallet.NewGeneratedWallet("user1", s.spec.WalletConfig)
	user1 := cosmosutil.NewInteractingWallet(s.chain, wallet1, cosmosutil.EncodingConfig{
		InterfaceRegistry: encodingConfig.InterfaceRegistry,
		Codec:             encodingConfig.Codec,
		TxConfig:          encodingConfig.TxConfig,
	})

	s.Require().NoError(err)
	s.user1 = user1

	wallet2, err := wallet.NewGeneratedWallet("user2", s.spec.WalletConfig)
	user2 := cosmosutil.NewInteractingWallet(s.chain, wallet2, cosmosutil.EncodingConfig{
		InterfaceRegistry: encodingConfig.InterfaceRegistry,
		Codec:             encodingConfig.Codec,
		TxConfig:          encodingConfig.TxConfig,
	})

	s.Require().NoError(err)
	s.user2 = user2

	wallet3, err := wallet.NewGeneratedWallet("user3", s.spec.WalletConfig)
	user3 := cosmosutil.NewInteractingWallet(s.chain, wallet3, cosmosutil.EncodingConfig{
		InterfaceRegistry: encodingConfig.InterfaceRegistry,
		Codec:             encodingConfig.Codec,
		TxConfig:          encodingConfig.TxConfig,
	})

	s.Require().NoError(err)
	s.user3 = user3

	err = s.chain.Init(ctx)
	s.Require().NoError(err)

	time.Sleep(2 * time.Second)

	s.chain.WaitForBlocks(ctx, 2)

	interactingFaucet := cosmosutil.NewInteractingWallet(s.chain, s.chain.GetFaucetWallet(), cosmosutil.EncodingConfig{
		InterfaceRegistry: encodingConfig.InterfaceRegistry,
		Codec:             encodingConfig.Codec,
		TxConfig:          encodingConfig.TxConfig,
	})

	sendGasSettings := petritypes.GasSettings{
		Gas:         100000,
		PricePerGas: int64(0),
		GasDenom:    s.spec.Denom,
	}

	txResp, err := s.chainClient.BankSend(ctx, *interactingFaucet, s.user1.Address(), sdk.NewCoins(sdk.NewCoin(s.chain.GetConfig().Denom, sdkmath.NewInt(1000000000))), sendGasSettings, true)
	fmt.Println(txResp)
	s.Require().NoError(err)

	_, err = s.chainClient.BankSend(ctx, *interactingFaucet, s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.chain.GetConfig().Denom, sdkmath.NewInt(10000000))), sendGasSettings, true)
	s.Require().NoError(err)

	_, err = s.chainClient.BankSend(ctx, *interactingFaucet, s.user3.Address(), sdk.NewCoins(sdk.NewCoin(s.chain.GetConfig().Denom, sdkmath.NewInt(10000000))), sendGasSettings, true)
	s.Require().NoError(err)

	// create the fuzzusers
	fuzzusers := make([]*cosmosutil.InteractingWallet, 0)
	for i := 0; i < 10; i++ {
		fuzzwallet, err := wallet.NewGeneratedWallet(fmt.Sprintf("fuzzuser%d", i), s.spec.WalletConfig)
		fuzzuser := cosmosutil.NewInteractingWallet(s.chain, fuzzwallet, cosmosutil.EncodingConfig{
			InterfaceRegistry: encodingConfig.InterfaceRegistry,
			Codec:             encodingConfig.Codec,
			TxConfig:          encodingConfig.TxConfig,
		})

		s.Require().NoError(err)

		_, err = s.chainClient.BankSend(ctx, *interactingFaucet, fuzzuser.Address(), sdk.NewCoins(sdk.NewCoin(s.chain.GetConfig().Denom, sdkmath.NewInt(1000000))), sendGasSettings, true)
		s.Require().NoError(err)

		fuzzusers = append(fuzzusers, fuzzuser)
	}
	s.fuzzusers = fuzzusers

	s.chain.WaitForBlocks(ctx, 5)
}

func (s *E2ETestSuite) TearDownSuite() {
	ctx := context.Background()

	s.chain.WaitForBlocks(ctx, 5) // give time to ingest latest metrics

	grafanaIP, err := s.grafanaTask.GetExternalAddress(ctx, "3000/tcp")
	s.Require().NoError(err)

	snapshot, err := monitoring.SnapshotGrafanaDashboard(ctx, "b8ff6e6f-5b4b-4d5e-bc50-91bbbf10f436", fmt.Sprintf("http://%s", grafanaIP))
	s.Require().NoError(err)

	// err = s.chain.Teardown(ctx)
	// s.Require().NoError(err)
	//
	// err = s.provider.Teardown(ctx)
	// s.Require().NoError(err)

	fmt.Printf("Grafana snapshot: %#v\n", snapshot)
}

func (s *E2ETestSuite) TestLoadTest() {
	var endpoints []string

	for _, val := range s.chain.GetValidators() {
		endpoint, err := val.GetTMClient(context.Background())
		s.Require().NoError(err)

		url := strings.Replace(endpoint.Remote(), "http", "ws", -1)

		endpoints = append(endpoints, fmt.Sprintf("%s/websocket", url))
	}

	ec := AMakeEncodingConfig()

	cf, err := petriload.NewDefaultClientFactory(
		petriload.ClientFactoryConfig{
			Chain:                 s.chain,
			Seeder:                s.user1,
			WalletConfig:          s.spec.WalletConfig,
			AmountToSend:          1000000,
			SkipSequenceIncrement: false,
			EncodingConfig: cosmosutil.EncodingConfig{
				InterfaceRegistry: ec.InterfaceRegistry,
				Codec:             ec.Codec,
				TxConfig:          ec.TxConfig,
			},
			MsgGenerator: func(sender []byte) ([]sdk.Msg, petritypes.GasSettings, error) {
				return []sdk.Msg{
						banktypes.NewMsgSend(
							sdk.AccAddress(sender),
							sdk.AccAddress([]byte{1}),
							sdk.NewCoins(sdk.NewCoin(s.chain.GetConfig().Denom, sdkmath.NewInt(1))),
						),
					}, petritypes.GasSettings{
						Gas:         200000,
						PricePerGas: int64(0),
						GasDenom:    s.chain.GetConfig().Denom,
					}, nil
			},
		}, app.ModuleBasics)

	s.Require().NoError(err)

	err = loadtest.RegisterClientFactory("cosmos_bank_send", cf)
	s.Require().NoError(err)

	cfg := loadtest.Config{
		ClientFactory:        "cosmos_bank_send",
		Connections:          1,
		Endpoints:            endpoints,
		Time:                 60,
		SendPeriod:           1,
		Rate:                 350,
		Size:                 250,
		Count:                -1,
		BroadcastTxMethod:    "async",
		EndpointSelectMethod: "supplied",
		StatsOutputFile:      "./test.csv",
	}

	err = loadtest.ExecuteStandalone(cfg)
	s.Require().NoError(err)
}

func (s *E2ETestSuite) DoNotTestValidBids() {
	params := s.QueryAuctionParams(s.T(), context.Background())
	escrowAddr := sdk.AccAddress(params.EscrowAccountAddress).String()

	s.Run("Valid Auction Bid", func() {
		// get escrow account balance before bid
		escrowAcctBalanceBeforeBid, err := s.chainClient.Balance(context.Background(), escrowAddr, params.ReserveFee.Denom)
		s.Require().NoError(err)

		// create bundle w/ a single tx
		// create message send tx
		tx := banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin(s.denom, sdkmath.NewInt(10000))))

		height, err := s.chain.Height(context.Background())
		require.NoError(s.T(), err)
		nextBlockHeight := height + 1

		fmt.Println("nextBlockHeight: ", nextBlockHeight)

		// create the MsgAuctioBid
		bidAmt := params.ReserveFee
		bid, bundledTxs := s.CreateAuctionBidMsg(context.Background(), s.user1, bidAmt, []Tx{
			{
				User: s.user1,
				Msgs: []sdk.Msg{
					tx,
				},
				SequenceIncrement: 1,
				Height:            nextBlockHeight,
			},
		})

		// broadcast + wait for the tx to be included in a block
		bidTx, err := s.user1.CreateSignedTx(context.Background(), 25000000, sdk.NewCoins(sdk.NewCoin(s.chain.GetConfig().Denom, sdkmath.NewInt(10))), nextBlockHeight, "", bid)
		s.Require().NoError(err)
		bidBz, err := s.chain.GetTxConfig().TxEncoder()(bidTx)
		s.Require().NoError(err)
		resp, err := s.user1.BroadcastTx(context.Background(), bidTx, true)

		if err != nil {
			s.T().Logf("error broadcasting tx: %s\ntx response: %v\n", err.Error(), resp)
		}
		s.Require().NoError(err)

		err = s.chain.WaitForHeight(context.Background(), nextBlockHeight)
		s.Require().NoError(err)

		// verify the block
		expectedBlock := [][]byte{
			bidBz,
			bundledTxs[0],
		}
		VerifyBlockWithExpectedBlock(s.T(), context.Background(), s.chainClient, nextBlockHeight, expectedBlock)

		// ensure that the escrow account has the correct balance
		escrowAcctBalanceAfterBid, err := s.chainClient.Balance(context.Background(), escrowAddr, params.ReserveFee.Denom)
		s.Require().NoError(err)

		expectedIncrement := escrowAddressIncrement(bidAmt.Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid.AddAmount(sdkmath.NewInt(expectedIncrement)), escrowAcctBalanceAfterBid)
	})
}

func (s *E2ETestSuite) QueryAuctionParams(t *testing.T, ctx context.Context) auctiontypes.Params {
	cc, err := s.chain.GetGRPCClient(ctx)
	defer cc.Close()

	auctionclient := auctiontypes.NewQueryClient(cc)

	res, err := auctionclient.Params(ctx, &auctiontypes.QueryParamsRequest{})
	require.NoError(t, err)

	return res.Params
}

func (s *E2ETestSuite) CreateAuctionBidMsg(ctx context.Context, searcher *cosmosutil.InteractingWallet, bid sdk.Coin, txsPerUser []Tx) (*auctiontypes.MsgAuctionBid, [][]byte) {
	// for each MessagesForUser get the signed bytes
	txs := make([][]byte, len(txsPerUser))
	for i, tx := range txsPerUser {
		acc, err := tx.User.Account(ctx)
		s.Require().NoError(err)

		gasPrice := sdkmath.NewInt(tx.GasPrice).Mul(sdkmath.NewInt(250000000))
		createdTx, err := tx.User.CreateTx(ctx, 25000000, sdk.NewCoins(sdk.NewCoin(s.chain.GetConfig().Denom, gasPrice)), tx.Height, "", tx.Msgs...)
		s.Require().NoError(err)

		signedTx, err := tx.User.SignTx(ctx, createdTx, acc.GetAccountNumber(), acc.GetSequence()+tx.SequenceIncrement)
		s.Require().NoError(err)

		bz, err := s.chain.GetTxConfig().TxEncoder()(signedTx)
		s.Require().NoError(err)

		txs[i] = bz
	}

	bech32SearcherAddress := searcher.FormattedAddress()
	accAddr, err := sdk.AccAddressFromBech32(bech32SearcherAddress)
	s.Require().NoError(err)

	// create a message auction bid
	return auctiontypes.NewMsgAuctionBid(
		accAddr,
		bid,
		txs,
	), txs
}

func escrowAddressIncrement(bid sdkmath.Int, proposerFee sdkmath.LegacyDec) int64 {
	return bid.Sub(sdkmath.LegacyNewDecFromInt(bid).Mul(proposerFee).RoundInt()).Int64()
}

type Tx struct {
	User               *cosmosutil.InteractingWallet
	Msgs               []sdk.Msg
	GasPrice           int64
	SequenceIncrement  uint64
	Height             uint64
	SkipInclusionCheck bool
	ExpectFail         bool
	IgnoreChecks       bool
}

func VerifyBlockWithExpectedBlock(t *testing.T, ctx context.Context, chainClient *cosmosutil.ChainClient, height uint64, txs [][]byte) {
	intHeight := int64(height)
	block, err := chainClient.Block(ctx, &intHeight)
	require.NoError(t, err)

	blockTxs := block.Block.Data.Txs

	t.Logf("verifying block %d", height)
	require.Equal(t, len(txs), len(blockTxs))
	for i, tx := range txs {
		t.Logf("verifying tx %d; expected %s, got %s", i, TxHash(tx), TxHash(blockTxs[i]))
		require.Equal(t, TxHash(tx), TxHash(blockTxs[i]))
	}
}

func TxHash(tx []byte) string {
	return strings.ToUpper(hex.EncodeToString(comettypes.Tx(tx).Hash()))
}
