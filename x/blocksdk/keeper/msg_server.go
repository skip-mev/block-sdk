package keeper

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/block-sdk/x/blocksdk/types"
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

func (m MsgServer) RegisterLane(goCtx context.Context, msg *types.MsgRegisterLane) (*types.MsgRegisterLaneResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// ensure that the message signer is the authority
	if msg.Authority != m.Keeper.GetAuthority() {
		return nil, fmt.Errorf("this message can only be executed by the authority; expected %s, got %s", m.Keeper.GetAuthority(), msg.Authority)
	}

	_, err := m.GetLane(ctx, msg.Lane.Id)
	if err == nil {
		return nil, fmt.Errorf("lane with ID %s is already registered", msg.Lane.Id)
	}

	m.SetLane(ctx, msg.Lane)

	// TODO emit event

	return nil, nil
}

func (m MsgServer) UpdateLane(goCtx context.Context, msg *types.MsgUpdateLane) (*types.MsgUpdateLaneResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// ensure that the message signer is the authority
	if msg.Authority != m.Keeper.GetAuthority() {
		return nil, fmt.Errorf("this message can only be executed by the authority; expected %s, got %s", m.Keeper.GetAuthority(), msg.Authority)
	}

	_, err := m.GetLane(ctx, msg.Lane.Id)
	if err != nil {
		return nil, fmt.Errorf("lane with ID %s not found", msg.Lane.Id)
	}

	m.SetLane(ctx, msg.Lane)

	// TODO emit event

	return nil, nil
}
