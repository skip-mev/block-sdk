package base_test

import (
	"math/rand"
	"testing"

<<<<<<< HEAD
	testutils "github.com/skip-mev/block-sdk/testutils"
	"github.com/stretchr/testify/suite"
=======
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	testutils "github.com/skip-mev/block-sdk/testutils"
	"github.com/skip-mev/block-sdk/x/auction/types"
>>>>>>> cbc0483 (chore(verifytx): Updating VerifyTx to Cache between Transactions (#137))
)

type BaseTestSuite struct {
	suite.Suite

	ctx            sdk.Context
	encodingConfig testutils.EncodingConfig
	random         *rand.Rand
	accounts       []testutils.Account
	gasTokenDenom  string
}

func TestBaseTestSuite(t *testing.T) {
	suite.Run(t, new(BaseTestSuite))
}

func (s *BaseTestSuite) SetupTest() {
	// Set up basic TX encoding config.
	s.encodingConfig = testutils.CreateTestEncodingConfig()

	// Create a few random accounts
	s.random = rand.New(rand.NewSource(1))
	s.accounts = testutils.RandomAccounts(s.random, 5)
	s.gasTokenDenom = "stake"

	key := storetypes.NewKVStoreKey(types.StoreKey)
	s.ctx = testutil.DefaultContext(key, storetypes.NewTransientStoreKey("transient_key"))
}
