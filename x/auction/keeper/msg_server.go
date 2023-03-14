package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/x/auction/types"
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

// AuctionBid is the server implementation for Msg/AuctionBid.
func (m MsgServer) AuctionBid(goCtx context.Context, msg *types.MsgAuctionBid) (*types.MsgAuctionBidResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// This should never return an error because the address was validated when the message was ingressed.
	bidder, err := sdk.AccAddressFromBech32(msg.Bidder)
	if err != nil {
		return nil, err
	}

	// Ensure that the number of transactions is less than or equal to the maximum allowed.
	maxBundleSize, err := m.Keeper.GetMaxBundleSize(ctx)
	if err != nil {
		return nil, err
	}

	if uint32(len(msg.Transactions)) > maxBundleSize {
		return nil, fmt.Errorf("the number of transactions in the bid is greater than the maximum allowed; expected <= %d, got %d", maxBundleSize, len(msg.Transactions))
	}

	// Attempt to send the bid to the module account.
	if err := m.Keeper.bankkeeper.SendCoinsFromAccountToModule(ctx, bidder, types.ModuleName, msg.Bid); err != nil {
		return nil, err
	}

	// TODO: figure out how to handle payments to the escrow address.
	// Ref: https://github.com/skip-mev/pob/issues/11

	return &types.MsgAuctionBidResponse{}, nil
}

// UpdateParams is the server implementation for Msg/UpdateParams.
func (m MsgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Ensure that the message signer is the authority.
	if msg.Authority != m.Keeper.GetAuthority() {
		return nil, fmt.Errorf("this message can only be executed by the authority; expected %s, got %s", m.Keeper.GetAuthority(), msg.Authority)
	}

	// Update the parameters.
	if err := m.Keeper.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
