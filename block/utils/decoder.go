package utils

import (
	"fmt"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultMaxSize is the default maximum size of the cache.
var DefaultMaxSize uint64 = 500

// CacheTxDecoder wraps the sdk.TxDecoder and caches the decoded transactions with
// an LRU'esque cache. Each transaction is cached using the transaction's hash
// as the key. The cache is purged when the number of transactions in the cache
// exceeds the maximum size. The oldest transactions are removed first.
type CacheTxDecoder struct {
	mut sync.Mutex

	decoder     sdk.TxDecoder
	cache       map[string]sdk.Tx
	window      []string
	insertIndex int
	oldestIndex int
	maxSize     uint64
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
		window:      make([]string, DefaultMaxSize),
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
		window:      make([]string, maxSize),
		insertIndex: 0,
		oldestIndex: 0,
		maxSize:     maxSize,
	}, nil
}

// Decode decodes the transaction bytes into a sdk.Tx. It caches the decoded
// transaction using the transaction's hash as the key.
func (ctd *CacheTxDecoder) TxDecoder() sdk.TxDecoder {
	return func(txBytes []byte) (sdk.Tx, error) {
		ctd.mut.Lock()
		defer ctd.mut.Unlock()

		hash := TxHash(txBytes)
		if tx, ok := ctd.cache[hash]; ok {
			return tx, nil
		}

		tx, err := ctd.decoder(txBytes)
		if err != nil {
			return nil, err
		}

		// Purge the cache if necessary
		if uint64(len(ctd.cache)) >= ctd.maxSize {
			// Purge the oldest transaction
			entry := ctd.window[ctd.oldestIndex]
			delete(ctd.cache, entry)

			// Increment the oldest index
			ctd.oldestIndex++
			ctd.oldestIndex %= int(ctd.maxSize)
		}

		// Update the cache and window
		ctd.cache[hash] = tx
		ctd.window[ctd.insertIndex] = hash

		// Increment the insert index
		ctd.insertIndex++
		ctd.insertIndex %= int(ctd.maxSize)

		return tx, nil
	}
}

// Len returns the number of transactions in the cache.
func (ctd *CacheTxDecoder) Len() int {
	ctd.mut.Lock()
	defer ctd.mut.Unlock()

	return len(ctd.cache)
}

// Contains returns true if the cache contains the transaction with the given hash.
func (ctd *CacheTxDecoder) Contains(txBytes []byte) bool {
	ctd.mut.Lock()
	defer ctd.mut.Unlock()

	hash := TxHash(txBytes)
	_, ok := ctd.cache[hash]
	return ok
}
