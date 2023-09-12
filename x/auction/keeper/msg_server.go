package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
<<<<<<< HEAD:x/builder/keeper/msg_server.go
	"github.com/skip-mev/block-sdk/x/builder/types"
=======

	"github.com/skip-mev/block-sdk/x/auction/types"
>>>>>>> 3c6f319 (feat(docs): rename x/builder -> x/auction (#55)):x/auction/keeper/msg_server.go
)

var _ types.MsgServer = MsgServer{}

// MsgServer is the wrapper for the auction module's msg service.
type MsgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the auction MsgServer interface.
func NewMsgServerImpl(keeper Keeper) *MsgServer {
	return &MsgServer{Keeper: keeper}
}

func (m MsgServer) AuctionBid(goCtx context.Context, msg *types.MsgAuctionBid) (*types.MsgAuctionBidResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// This should never return an error because the address was validated when
	// the message was ingressed.
	bidder, err := sdk.AccAddressFromBech32(msg.Bidder)
	if err != nil {
		return nil, err
	}

	params, err := m.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	if uint32(len(msg.Transactions)) > params.MaxBundleSize {
		return nil, fmt.Errorf("the number of transactions in the bid is greater than the maximum allowed; expected <= %d, got %d", params.MaxBundleSize, len(msg.Transactions))
	}

	escrowAddress := params.EscrowAccountAddress

	var proposerReward sdk.Coins
	if params.ProposerFee.IsZero() {
		// send the entire bid to the escrow account when no proposer fee is set
		if err := m.bankKeeper.SendCoins(ctx, bidder, escrowAddress, sdk.NewCoins(msg.Bid)); err != nil {
			return nil, err
		}
	} else {
		rewardsAddress, err := m.rewardsAddressProvider.GetRewardsAddress(ctx)
		if err != nil {
			// In the case where the rewards address provider returns an error, the
			// escrow account will receive the entire bid.
			rewardsAddress = escrowAddress
		}

		// determine the amount of the bid that goes to the (previous) proposer
		bid := sdk.NewDecCoinsFromCoins(msg.Bid)
		proposerReward, _ = bid.MulDecTruncate(params.ProposerFee).TruncateDecimal()

		if err := m.bankKeeper.SendCoins(ctx, bidder, rewardsAddress, proposerReward); err != nil {
			return nil, err
		}

		// Determine the amount of the remaining bid that goes to the escrow account.
		// If a decimal remainder exists, it'll stay with the bidding account.
		escrowTotal := bid.Sub(sdk.NewDecCoinsFromCoins(proposerReward...))
		escrowReward, _ := escrowTotal.TruncateDecimal()

		if err := m.bankKeeper.SendCoins(ctx, bidder, escrowAddress, escrowReward); err != nil {
			return nil, err
		}
	}

	bundledTxHashes := make([]string, len(msg.Transactions))
	for i, refTxRaw := range msg.Transactions {
		hash := sha256.Sum256(refTxRaw)
		bundledTxHashes[i] = hex.EncodeToString(hash[:])
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeAuctionBid,
			sdk.NewAttribute(types.EventAttrBidder, msg.Bidder),
			sdk.NewAttribute(types.EventAttrBid, msg.Bid.String()),
			sdk.NewAttribute(types.EventAttrProposerReward, proposerReward.String()),
			sdk.NewAttribute(types.EventAttrBundledTxs, strings.Join(bundledTxHashes, ",")),
		),
	)

	return &types.MsgAuctionBidResponse{}, nil
}

func (m MsgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// ensure that the message signer is the authority
	if msg.Authority != m.Keeper.GetAuthority() {
		return nil, fmt.Errorf("this message can only be executed by the authority; expected %s, got %s", m.Keeper.GetAuthority(), msg.Authority)
	}

	if err := m.Keeper.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
