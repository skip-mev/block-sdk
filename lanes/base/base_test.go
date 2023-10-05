package base_test

import (
	"math/rand"
	"testing"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	testutils "github.com/skip-mev/block-sdk/testutils"
	"github.com/skip-mev/block-sdk/x/auction/types"
	"github.com/stretchr/testify/suite"
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
