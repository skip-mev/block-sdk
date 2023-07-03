package cli

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/x/builder/types"
	"github.com/spf13/cobra"
)

// NewTxCmd returns a root CLI command handler for all x/builder transaction
// commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Builder transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewAuctionBidTx(),
	)

	return txCmd
}

func NewAuctionBidTx() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auction-bid [bidder] [bid] [bundled_tx1,bundled_tx2,...,bundled_txN]",
		Short: "Create an auction bid transaction with signed bundled transactions",
		Long: `Create an auction bid transaction with a list of signed bundled transactions,
where each transaction is a hex-encoded string of a signed transaction.
`,
		Args:    cobra.ExactArgs(3),
		Example: "auction-bid cosmos1... 10000uatom 0xFF...,0xCC...,0xAA...",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Flags().Set(flags.FlagFrom, args[0]); err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			bid, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}

			// ensure timeout is non-zero
			timeoutHeight, _ := cmd.Flags().GetUint64(flags.FlagTimeoutHeight)
			if timeoutHeight == 0 {
				return errors.New("timeout height must be greater than 0")
			}

			tokens := strings.Split(args[2], ",")
			bundledTxs := make([][]byte, len(tokens))
			for i, token := range tokens {
				rawTx, err := hex.DecodeString(token)
				if err != nil {
					return fmt.Errorf("failed to HEX decode bundled transaction %d: %w", i, err)
				}

				bundledTxs[i] = rawTx
			}

			msg := types.NewMsgAuctionBid(clientCtx.GetFromAddress(), bid, bundledTxs)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
