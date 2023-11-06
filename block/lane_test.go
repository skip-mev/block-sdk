package block_test

import (
	"fmt"

	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/block/mocks"
)

func (suite *BlockBusterTestSuite) TestFindLane() {
	lanes := make([]block.Lane, 30)
	cleanup := func() {
		for i := range lanes {
			_ = lanes[i].Name()
		}
	}
	defer cleanup()

	for i := range lanes {
		laneMock := mocks.NewLane(suite.T())
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
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			gotLane, gotIndex, gotFound := block.FindLane(tc.args.lanes, tc.args.name)
			if tc.wantFound {
				suite.Require().True(gotFound)
				suite.Require().Equal(tc.wantLane, gotLane)
				suite.Require().Equal(tc.wantIndex, gotIndex)
				return
			}

			suite.Require().False(gotFound)
		})
	}
}
