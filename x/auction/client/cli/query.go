package cli

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
<<<<<<< HEAD:x/builder/client/cli/query.go
	"github.com/skip-mev/block-sdk/x/builder/types"
	"github.com/spf13/cobra"
=======
	"github.com/spf13/cobra"

	"github.com/skip-mev/block-sdk/x/auction/types"
>>>>>>> 3c6f319 (feat(docs): rename x/builder -> x/auction (#55)):x/auction/client/cli/query.go
)

// GetQueryCmd returns the cli query commands for the auction module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdQueryParams(),
	)

	return cmd
}

// CmdQueryParams implements a command that will return the current parameters of the auction module.
func CmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the current parameters of the auction module",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			request := &types.QueryParamsRequest{}
			response, err := queryClient.Params(context.Background(), request)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(&response.Params)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
