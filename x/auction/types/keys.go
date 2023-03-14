package types

const (
	// ModuleName is the name of the auction module
	ModuleName = "auction"

	// StoreKey is the default store key for the auction module
	StoreKey = ModuleName

	// RouterKey is the message route for the auction module
	RouterKey = ModuleName

	// QuerierRoute is the querier route for the auction module
	QuerierRoute = ModuleName
)

const (
	prefixParams = iota + 1
)

// KeyParams is the store key for the auction module's parameters.
var KeyParams = []byte{prefixParams}
