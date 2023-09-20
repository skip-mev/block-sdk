package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/skip-mev/block-sdk/block"
	"github.com/skip-mev/block-sdk/block/utils"
	auctionante "github.com/skip-mev/block-sdk/x/auction/ante"
	auctionkeeper "github.com/skip-mev/block-sdk/x/auction/keeper"
)

type AnteHandlerOptions struct {
	BaseOptions   ante.HandlerOptions
	Mempool       block.Mempool
	MEVLane       auctionante.MEVLane
	TxDecoder     sdk.TxDecoder
	TxEncoder     sdk.TxEncoder
	auctionkeeper auctionkeeper.Keeper
	FreeLane      block.Lane
}

// NewPOBAnteHandler wraps all of the default Cosmos SDK AnteDecorators with the POB AnteHandler.
func NewAnteHandler(options AnteHandlerOptions) sdk.AnteHandler {
	if options.BaseOptions.AccountKeeper == nil {
		panic("account keeper is required for ante builder")
	}

	if options.BaseOptions.BankKeeper == nil {
		panic("bank keeper is required for ante builder")
	}

	if options.BaseOptions.SignModeHandler == nil {
		panic("sign mode handler is required for ante builder")
	}

	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		ante.NewExtensionOptionsDecorator(options.BaseOptions.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.BaseOptions.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.BaseOptions.AccountKeeper),
		utils.NewIgnoreDecorator(
			ante.NewDeductFeeDecorator(
				options.BaseOptions.AccountKeeper,
				options.BaseOptions.BankKeeper,
				options.BaseOptions.FeegrantKeeper,
				options.BaseOptions.TxFeeChecker,
			),
			options.FreeLane,
		),
		ante.NewSetPubKeyDecorator(options.BaseOptions.AccountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(options.BaseOptions.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.BaseOptions.AccountKeeper, options.BaseOptions.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.BaseOptions.AccountKeeper, options.BaseOptions.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.BaseOptions.AccountKeeper),
		auctionante.NewAuctionDecorator(options.auctionkeeper, options.TxEncoder, options.MEVLane, options.Mempool),
	}

	return sdk.ChainAnteDecorators(anteDecorators...)
}
