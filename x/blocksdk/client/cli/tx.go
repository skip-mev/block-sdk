package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/skip-mev/block-sdk/x/blocksdk/types"
	"github.com/spf13/cobra"
)

// NewTxCmd returns a root CLI command handler for all x/blocksdk transaction
// commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "blocksdk transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand()

	return txCmd
}
