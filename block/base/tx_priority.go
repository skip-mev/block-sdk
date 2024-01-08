package base

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Coins map[string]math.Int

// DefaultTxPriority returns a default implementation of the TxPriority. It prioritizes
// transactions by their fee.
func DefaultTxPriority() TxPriority[string] {
	return TxPriority[string]{
		GetTxPriority: func(goCtx context.Context, tx sdk.Tx) string {
			feeTx, ok := tx.(sdk.FeeTx)
			if !ok {
				return ""
			}

			return coinsToString(feeTx.GetFee())
		},
		Compare: func(a, b string) int {
			aCoins, _ := coinsFromString(a)
			bCoins, _ := coinsFromString(b)

			switch {
			case aCoins.Greater(bCoins):
				return 1

			case bCoins.Greater(aCoins):
				return -1

			default:
				return 0
			}
		},
		MinValue: "",
	}
}

func coinsToString(coins sdk.Coins) string {
	// sort the coins by denomination
	coins.Sort()

	// to avoid dealing with regex, etc. we use a , to separate denominations from amounts
	// e.g. 10000,stake,10000,atom
	coinString := ""
	for i, coin := range coins {
		coinString += coin.Amount.String() + "," + coin.Denom
		if i != len(coins)-1 {
			coinString += ","
		}
	}

	return coinString
}

// coinsFromString converts a string of coins to a sdk.Coins object.
func coinsFromString(coinsString string) (Coins, error) {
	// if its empty string (zero value), we return nil
	if coinsString == "" {
		return nil, nil
	}

	// split the string by commas
	coinStrings := strings.Split(coinsString, ",")

	// if the length is odd, then the given string is invalid
	if len(coinStrings)%2 != 0 {
		return nil, fmt.Errorf("invalid coins string: %s", coinsString)
	}

	coins := make(Coins, len(coinsString)/2)
	for i := 0; i < len(coinStrings); i += 2 {
		// split the string by pipe
		amount, ok := intFromString(coinStrings[i])
		if !ok {
			return nil, fmt.Errorf("invalid amount: %s, denom: %s", coinStrings[i], coinStrings[i+1])
		}

		coins[coinStrings[i+1]] = amount
	}

	return coins, nil
}

func intFromString(str string) (math.Int, bool) {
	// first attempt to get int64 from the string
	int64Val, err := strconv.ParseInt(str, 10, 64)
	if err == nil {
		return math.NewInt(int64Val), true
	}

	// if we can't get an int64, then get raw math.Int
	return math.NewIntFromString(str)
}

// Greater returns true if lhs is strictly greater than rhs, and false otherwise. Notice, lhs / rhs must be comparable,
// specifically, they must have the exact same denoms, otherwise, they aren't comparable.
func (lhs Coins) Greater(rhs Coins) bool {
	// if a or b is nil, then return whether a is non-nil
	if lhs == nil || rhs == nil {
		return lhs != nil
	}

	// for each of a's denoms, check if b has the same denom
	if len(lhs) != len(rhs) {
		return false
	}

	// for each of a's denoms, check if a is greater
	for denom, aAmount := range lhs {
		// b does not have the corresponding denom, a is not greater
		bAmount, ok := rhs[denom]
		if !ok {
			return false
		}

		// a is not greater than b
		if !aAmount.GT(bAmount) {
			return false
		}
	}

	return true
}

// DeprecatedTxPriority serves the same purpose as DefaultTxPriority, however, it is significantly slower- on the order of
// 6-10x slower.
func DeprecatedTxPriority() TxPriority[string] {
	return TxPriority[string]{
		GetTxPriority: func(goCtx context.Context, tx sdk.Tx) string {
			feeTx, ok := tx.(sdk.FeeTx)
			if !ok {
				return ""
			}

			return feeTx.GetFee().String()
		},
		Compare: func(a, b string) int {
			aCoins, _ := sdk.ParseCoinsNormalized(a)
			bCoins, _ := sdk.ParseCoinsNormalized(b)

			switch {
			case aCoins == nil && bCoins == nil:
				return 0

			case aCoins == nil:
				return -1

			case bCoins == nil:
				return 1

			default:
				switch {
				case aCoins.IsAllGT(bCoins):
					return 1

				case aCoins.IsAllLT(bCoins):
					return -1

				default:
					return 0
				}
			}
		},
		MinValue: "",
	}
}
