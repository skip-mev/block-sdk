package types

const (
	// ModuleName is the name of the blocksdk module
	ModuleName = "blocksdk"

	// StoreKey is the default store key for the blocksdk module
	StoreKey = ModuleName

	// RouterKey is the message route for the blocksdk module
	RouterKey = ModuleName

	// QuerierRoute is the querier route for the blocksdk module
	QuerierRoute = ModuleName
)

const (
	prefixLanes = iota
	prefixParams
)

var (
	// KeyLanes is the store key for the lanes.
	KeyLanes = []byte{prefixLanes}
	// KeyParams is the store key for the params.
	KeyParams = []byte{prefixParams}
)
