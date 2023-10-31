package keeper

import (
	"context"
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

func (m MsgServer) RegisterLane(goCtx context.Context, _ *types.MsgRegisterLane) (*types.MsgRegisterLaneResponse, error) {
	// TODO

	return nil, nil
}

func (m MsgServer) UpdateLane(goCtx context.Context, _ *types.MsgUpdateLane) (*types.MsgUpdateLaneResponse, error) {
	// TODO

	return nil, nil
}
