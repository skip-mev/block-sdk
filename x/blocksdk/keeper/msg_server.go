package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/x/blocksdk/types"
)

var _ types.MsgServer = MsgServer{}

// MsgServer is the wrapper for the x/blocksdk module's msg service.
type MsgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the x/blocksdk MsgServer interface.
func NewMsgServerImpl(keeper Keeper) *MsgServer {
	return &MsgServer{Keeper: keeper}
}

// UpdateLane implements the message service for updating a lane that exists in the store.
func (m MsgServer) UpdateLane(goCtx context.Context, msg *types.MsgUpdateLane) (*types.MsgUpdateLaneResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// ensure that the message signer is the authority
	if msg.Authority != m.Keeper.GetAuthority() {
		return nil, fmt.Errorf("this message can only be executed by the authority; expected %s, got %s", m.Keeper.GetAuthority(), msg.Authority)
	}

	oldLane, err := m.GetLane(ctx, msg.Lane.Id)
	if err != nil {
		return nil, fmt.Errorf("lane with ID %s not found", msg.Lane.Id)
	}

	if oldLane.Order != msg.Lane.Order {
		return nil, fmt.Errorf("unable to change lane order using UpdateLane method")
	}

	m.setLane(ctx, msg.Lane)

	// TODO emit event

	return nil, nil
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
