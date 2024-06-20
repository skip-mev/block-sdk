package utils

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	// DefaultMaxSize is the default maximum size of the cache.
	DefaultMaxSize uint64 = 500
)

// CacheTxDecoder wraps the sdk.TxDecoder and caches the decoded transactions with
// an LRU'esque cache. Each transaction is cached using the transaction's hash
// as the key. The cache is purged when the number of transactions in the cache
// exceeds the maximum size. The oldest transactions are removed first.
type CacheTxDecoder struct {
	decoder     sdk.TxDecoder
	cache       map[string]sdk.Tx
	timestamps  []CacheValue
	insertIndex int
	oldestIndex int
	maxSize     uint64
}

// CacheValue is a wrapper struct for the cached transaction along with the
// timestamp of when it was added to the cache. This is used to determine the
// oldest transaction in the cache.
type CacheValue struct {
	hash      string
	timestamp time.Time
}

// NewDefaultCacheTxDecoder returns a new CacheTxDecoder.
func NewDefaultCacheTxDecoder(
	decoder sdk.TxDecoder,
) (*CacheTxDecoder, error) {
	if decoder == nil {
		return nil, fmt.Errorf("decoder cannot be nil")
	}

	return &CacheTxDecoder{
		decoder:     decoder,
		cache:       make(map[string]sdk.Tx),
		timestamps:  make([]CacheValue, DefaultMaxSize),
		insertIndex: 0,
		oldestIndex: 0,
		maxSize:     DefaultMaxSize,
	}, nil
}

// NewCacheTxDecoder returns a new CacheTxDecoder with the given cache interval.
func NewCacheTxDecoder(
	decoder sdk.TxDecoder,
	maxSize uint64,
) (*CacheTxDecoder, error) {
	if decoder == nil {
		return nil, fmt.Errorf("decoder cannot be nil")
	}

	return &CacheTxDecoder{
		decoder:     decoder,
		cache:       make(map[string]sdk.Tx),
		timestamps:  make([]CacheValue, maxSize),
		insertIndex: 0,
		oldestIndex: 0,
		maxSize:     maxSize,
	}, nil
}

// Decode decodes the transaction bytes into a sdk.Tx. It caches the decoded
// transaction using the transaction's hash as the key.
func (ctd *CacheTxDecoder) TxDecoder() sdk.TxDecoder {
	return func(txBytes []byte) (sdk.Tx, error) {
		hash := TxHash(txBytes)
		if tx, ok := ctd.cache[hash]; ok {
			return tx, nil
		}

		// Purge the cache if necessary
		if uint64(len(ctd.cache)) >= ctd.maxSize {
			// Purge the oldest transaction
			entry := ctd.timestamps[ctd.oldestIndex]
			delete(ctd.cache, entry.hash)

			// Increment the oldest index
			ctd.oldestIndex++
			ctd.oldestIndex = ctd.oldestIndex % int(ctd.maxSize)
		}

		tx, err := ctd.decoder(txBytes)
		if err != nil {
			return nil, err
		}

		// Update the cache
		ctd.cache[hash] = tx

		// Add the hash to the timestamps slice
		ctd.timestamps[ctd.insertIndex] = CacheValue{
			hash:      hash,
			timestamp: time.Now(),
		}

		// Increment the insert index
		ctd.insertIndex++
		ctd.insertIndex = ctd.insertIndex % int(ctd.maxSize)

		return tx, nil
	}
}

// Len returns the number of transactions in the cache.
func (ctd *CacheTxDecoder) Len() int {
	return len(ctd.cache)
}
