package block_test

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/skip-mev/block-sdk/block/mocks"

	"github.com/skip-mev/block-sdk/block"
)

func TestFindLane(t *testing.T) {
	lanes := make([]block.Lane, 30)
	cleanup := func() {
		for i := range lanes {
			_ = lanes[i].Name()
		}
	}
	defer cleanup()

	for i := range lanes {
		laneMock := mocks.NewLane(t)
		laneMock.On("Name").Return(fmt.Sprintf("lane%d", i))
		lanes[i] = laneMock
	}

	type args struct {
		lanes []block.Lane
		name  string
	}
	tests := []struct {
		name      string
		args      args
		wantLane  block.Lane
		wantIndex int
		wantFound bool
	}{
		{
			name: "invalid lane not found",
			args: args{
				lanes: lanes,
				name:  "invalid",
			},
			wantFound: false,
		},
		{
			name: "valid lane1",
			args: args{
				lanes: lanes,
				name:  "lane1",
			},
			wantLane:  lanes[1],
			wantIndex: 1,
			wantFound: true,
		},
		{
			name: "valid lane15",
			args: args{
				lanes: lanes,
				name:  "lane15",
			},
			wantLane:  lanes[15],
			wantIndex: 15,
			wantFound: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLane, gotIndex, gotFound := block.FindLane(tt.args.lanes, tt.args.name)
			if tt.wantFound {
				require.True(t, gotFound)
				require.Equal(t, tt.wantLane, gotLane)
				require.Equal(t, tt.wantIndex, gotIndex)
				return
			}

			require.False(t, gotFound)

		})
	}
}
