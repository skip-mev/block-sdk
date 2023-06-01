package base

import sdk "github.com/cosmos/cosmos-sdk/types"

func (l *DefaultLane) PrepareLane(sdk.Context, int64, map[string][]byte) ([][]byte, error) {
	panic("implement me")
}

func (l *DefaultLane) ProcessLane(sdk.Context, [][]byte) error {
	panic("implement me")
}

func (l *DefaultLane) VerifyTx(sdk.Context, sdk.Tx) error {
	panic("implement me")
}
