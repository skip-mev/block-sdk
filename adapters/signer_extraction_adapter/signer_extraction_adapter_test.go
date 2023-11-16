package signerextraction_test

import (
	"math/rand"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/suite"

	signer_extraction "github.com/skip-mev/block-sdk/adapters/signer_extraction_adapter"
	testutils "github.com/skip-mev/block-sdk/testutils"
)

type SignerExtractionAdapterTestSuite struct {
	suite.Suite
	txConfig client.TxConfig
	accts    []testutils.Account
	adapter  signer_extraction.DefaultAdapter
}

func TestSignerExtractionAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(SignerExtractionAdapterTestSuite))
}

func (s *SignerExtractionAdapterTestSuite) SetupTest() {
	encodingConfig := testutils.CreateTestEncodingConfig()
	s.txConfig = encodingConfig.TxConfig

	accts := testutils.RandomAccounts(rand.New(rand.NewSource(1)), 2)

	s.accts = accts
}

func (s *SignerExtractionAdapterTestSuite) TestGetSigners() {
	acct := s.accts[0]
	tx, err := testutils.CreateTx(s.txConfig, acct, 1, 1, []sdk.Msg{
		&banktypes.MsgSend{
			FromAddress: acct.Address.String(),
			ToAddress:   acct.Address.String(),
			Amount:      sdk.NewCoins(sdk.NewInt64Coin("test", 1)),
		},
	}, sdk.NewCoins(sdk.NewCoin("test", math.NewInt(1)))...)
	s.Require().NoError(err)

	signers, err := s.adapter.GetSigners(tx)
	s.Require().NoError(err)

	s.Require().Len(signers, 1)
	s.Require().Equal(acct.Address.String(), signers[0].Signer.String())
	s.Require().Equal(uint64(1), signers[0].Sequence)
}
