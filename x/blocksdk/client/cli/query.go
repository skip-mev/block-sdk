package cli

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"

	"github.com/skip-mev/block-sdk/x/blocksdk/types"
)

// GetQueryCmd returns the cli query commands for the blocksdk module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdQueryLane(),
		CmdQueryLanes(),
	)

	return cmd
}

// CmdQueryLane implements a command that will return a lane by its ID.
func CmdQueryLane() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lane",
		Short: "Query the a lane by its ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			request := &types.QueryLaneRequest{}
			response, err := queryClient.Lane(clientCtx.CmdContext, request)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(&response.Lane)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// CmdQueryLanes implements a command that will return all lanes.
func CmdQueryLanes() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lanes",
		Short: "Query the all lanes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			request := &types.QueryLanesRequest{}
			response, err := queryClient.Lanes(clientCtx.CmdContext, request)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(response)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
