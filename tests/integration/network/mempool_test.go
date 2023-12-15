package integration_test

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/math"
	cmthttp "github.com/cometbft/cometbft/rpc/client/http"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"google.golang.org/grpc"

	blockservicetypes "github.com/skip-mev/block-sdk/block/service/types"
	testutils "github.com/skip-mev/block-sdk/testutils/networksuite"
	auctiontypes "github.com/skip-mev/block-sdk/x/auction/types"
)

const (
	free = "free"
	base = "base"
	mev  = "mev"
)

var cdc *codec.ProtoCodec

func init() {
	ir := codectypes.NewInterfaceRegistry()

	authtypes.RegisterInterfaces(ir)
	cryptocodec.RegisterInterfaces(ir)
	cdc = codec.NewProtoCodec(ir)
}

// TestLanedMempool tests that the block-sdk mempool is properly synced w/ comet's mempool
func (s *NetworkTestSuite) TestLanedMempoolSyncWithComet() {
	cc, closefn, err := s.NetworkSuite.GetGRPC()
	s.Require().NoError(err)
	defer closefn()

	tmClient, err := s.GetTMClient()
	s.Require().NoError(err)

	blockClient := blockservicetypes.NewServiceClient(cc)

	ctx, closeCtx := context.WithTimeout(context.Background(), 1*time.Minute)
	defer closeCtx()
	val, err := getFirstValidator(ctx, stakingtypes.NewQueryClient(cc))
	s.Require().NoError(err)
	acc := *s.Accounts[0]

	s.Run("test free-lane sync", func() {
		s.Run("all valid txs", func() {
			// create a bunch of delegation txs and check the app-mempool v. comet-mempool
			msg := createFreeTx(acc.Address(), val, sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(10)))
			s.Require().NoError(checkParity(ctx, tmClient, blockClient, cc, acc, free, msg))
		})

		s.Run("bid Verify invalidates later tx Verify", func() {
			// create a new account
			zeroAccount := testutils.NewAccount()

			// initialize the account w/ enough for a single tx
			send := banktypes.NewMsgSend(acc.Address(), zeroAccount.Address(), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(3000100))))

			seq, num, err := getAccount(ctx, authtypes.NewQueryClient(cc), acc)
			s.Require().NoError(err)

			// send the tx (pay for fees)
			tx, err := acc.CreateTx(ctx, num, seq, 1000000, 1000000, 1000000, send)
			s.Require().NoError(err)

			// commit tx
			res, err := tmClient.BroadcastTxCommit(ctx, tx)
			s.Require().NoError(err)
			s.Require().Equal(uint32(0), res.TxResult.Code)

			// create a delegation tx -> this should spend fees in zeroAccount
			msg := banktypes.NewMsgSend(zeroAccount.Address(), acc.Address(), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1))))

			// update the balance of zeroAccount to pay for the next tx
			updateMsg := banktypes.NewMsgSend(acc.Address(), zeroAccount.Address(), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1000000))))

			status, err := tmClient.Status(ctx)
			s.Require().NoError(err)
			nextHeight := uint64(status.SyncInfo.LatestBlockHeight) + 1

			// pay for fees of next tx for zeroAccount (account for bid sequence)
			tx2, err := acc.CreateTx(ctx, num, seq+2, 1000000, 1000000, nextHeight, updateMsg)
			s.Require().NoError(err)

			seq2, num2, err := getAccount(ctx, authtypes.NewQueryClient(cc), *zeroAccount)
			s.Require().NoError(err)

			// spends all funds in account on fee deduction -> fees are refilled after bid
			txToWrap, err := zeroAccount.CreateTx(ctx, num2, seq2, 1000000, 3000000, 3000000, msg)
			s.Require().NoError(err)

			// first delegate tx (just used to increment sequence)
			firstDelegateTx, err := zeroAccount.CreateTx(ctx, num2, seq2, 1000000, 1000000, 1000000, msg)
			s.Require().NoError(err)

			// ordered after bid, and shld fail in PrepareProposal as there will be no funds to pay (will be removed from lane)
			secondDelegateTx, err := zeroAccount.CreateTx(ctx, num2, seq2+1, 1000000, 1000000, 1000000, msg)
			s.Require().NoError(err)

			// create a bid wrapping firstDelegateTx, tx2 -> i.e spend funds in zeroAccount and refill
			bid := auctiontypes.NewMsgAuctionBid(acc.Address(), sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1000000)), [][]byte{txToWrap, tx2})
			bidTx, err := acc.CreateTx(ctx, num, seq+1, 1000000, 1000000, nextHeight, bid)
			s.Require().NoError(err)

			// broadcast txs
			resp, err := tmClient.BroadcastTxSync(ctx, bidTx)
			s.Require().NoError(err)
			s.Require().Equal(uint32(0), resp.Code)

			resp, err = tmClient.BroadcastTxSync(ctx, firstDelegateTx)
			s.Require().NoError(err)
			s.Require().Equal(uint32(0), resp.Code)

			resp, err = tmClient.BroadcastTxSync(ctx, secondDelegateTx)
			s.Require().NoError(err)
			s.Require().Equal(uint32(0), resp.Code)

			// wait for commit of bid
			s.Require().NoError(waitForTxCommit(ctx, tmClient, cmttypes.Tx(bidTx).Hash()))

			// check mempool size
			txs, err := tmClient.NumUnconfirmedTxs(ctx)
			s.Require().NoError(err)

			cmtTxs := uint64(txs.Total)

			// check app mempool size
			appTxDist, err := blockClient.GetTxDistribution(ctx, &blockservicetypes.GetTxDistributionRequest{})
			s.Require().NoError(err)

			// check parity
			appTxs := 0
			for _, tx := range appTxDist.Distribution {
				appTxs += int(tx)
			}

			s.Require().Equal(appTxs, int(cmtTxs))
		})
	})
}

func createFreeTx(delegator sdk.AccAddress, validator sdk.ValAddress, amount sdk.Coin) sdk.Msg {
	return stakingtypes.NewMsgDelegate(delegator.String(), validator.String(), amount)
}

func getFirstValidator(ctx context.Context, cc stakingtypes.QueryClient) (sdk.ValAddress, error) {
	resp, err := cc.Validators(ctx, &stakingtypes.QueryValidatorsRequest{})
	if err != nil {
		return nil, err
	}

	if len(resp.Validators) == 0 {
		return nil, nil
	}

	return sdk.ValAddressFromBech32(resp.Validators[0].OperatorAddress)
}

func getAccount(ctx context.Context, cc authtypes.QueryClient, acc testutils.Account) (uint64, uint64, error) {
	resp, err := cc.Account(ctx, &authtypes.QueryAccountRequest{Address: acc.Address().String()})
	if err != nil {
		return 0, 0, err
	}

	var accI sdk.AccountI
	if err := cdc.UnpackAny(resp.Account, &accI); err != nil {
		return 0, 0, err
	}

	return accI.GetSequence(), accI.GetAccountNumber(), nil
}

func checkParity(
	ctx context.Context, tmClient *cmthttp.HTTP, blockClient blockservicetypes.ServiceClient,
	cc *grpc.ClientConn, acc testutils.Account, lane string, msg sdk.Msg,
) error {
	seq, num, err := getAccount(ctx, authtypes.NewQueryClient(cc), acc)
	if err != nil {
		return err
	}

	res, err := tmClient.Status(ctx)
	if err != nil {
		return err
	}

	height := res.SyncInfo.LatestBlockHeight
	// send 100 txs to the app and check mempool sizes
	numTxs := 100
	txsCh := make(chan []byte, numTxs)
	done := make(chan struct{})

	// spin GR to wait on tx inclusions
	go func() {
		for tx := range txsCh {
			// check for tx inclusion
			waitForTxCommit(ctx, tmClient, tx)
		}
		close(done)
	}()

	for i := 0; i < numTxs; i++ {
		tx, err := acc.CreateTx(ctx, num, seq+uint64(i), 1000000, 1000000, uint64(height+10), msg)
		if err != nil {
			return err
		}

		res, err := tmClient.BroadcastTxSync(ctx, tx)
		if err != nil {
			return err
		}

		txsCh <- res.Hash
	}
	// all txs are sent
	close(txsCh)

	// wait for all txs to be included before checking size
	<-done

	// check comet mempool size
	txs, err := tmClient.NumUnconfirmedTxs(ctx)
	if err != nil {
		return err
	}

	cmtTxs := uint64(txs.Total)

	// check app mempool size
	appTxs, err := blockClient.GetTxDistribution(ctx, &blockservicetypes.GetTxDistributionRequest{})
	if err != nil {
		return err
	}

	if cmtTxs != appTxs.Distribution[lane] {
		return fmt.Errorf("mempool size mismatch: %d != %d", cmtTxs, appTxs.Distribution[free])
	}

	return nil
}

func waitForTxCommit(ctx context.Context, client *cmthttp.HTTP, hash []byte) error {
	_, err := client.Tx(ctx, hash, false)
	for ; err != nil; _, err = client.Tx(ctx, hash, false) {
		time.Sleep(time.Millisecond)
	}
	return nil
}
