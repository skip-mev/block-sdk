// Code generated by mockery v2.43.2. DO NOT EDIT.

package mocks

import (
	context "context"

	types "github.com/cosmos/cosmos-sdk/types"
	mock "github.com/stretchr/testify/mock"
)

// BankKeeper is an autogenerated mock type for the BankKeeper type
type BankKeeper struct {
	mock.Mock
}

// GetBalance provides a mock function with given fields: ctx, addr, denom
func (_m *BankKeeper) GetBalance(ctx context.Context, addr types.AccAddress, denom string) types.Coin {
	ret := _m.Called(ctx, addr, denom)

	if len(ret) == 0 {
		panic("no return value specified for GetBalance")
	}

	var r0 types.Coin
	if rf, ok := ret.Get(0).(func(context.Context, types.AccAddress, string) types.Coin); ok {
		r0 = rf(ctx, addr, denom)
	} else {
		r0 = ret.Get(0).(types.Coin)
	}

	return r0
}

// SendCoins provides a mock function with given fields: ctx, fromAddr, toAddr, amt
func (_m *BankKeeper) SendCoins(ctx context.Context, fromAddr types.AccAddress, toAddr types.AccAddress, amt types.Coins) error {
	ret := _m.Called(ctx, fromAddr, toAddr, amt)

	if len(ret) == 0 {
		panic("no return value specified for SendCoins")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.AccAddress, types.AccAddress, types.Coins) error); ok {
		r0 = rf(ctx, fromAddr, toAddr, amt)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewBankKeeper creates a new instance of BankKeeper. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBankKeeper(t interface {
	mock.TestingT
	Cleanup(func())
},
) *BankKeeper {
	mock := &BankKeeper{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
