// Code generated by mockery v2.30.1. DO NOT EDIT.

package mocks

import (
	types "github.com/cosmos/cosmos-sdk/types"
	mock "github.com/stretchr/testify/mock"
)

// RewardsAddressProvider is an autogenerated mock type for the RewardsAddressProvider type
type RewardsAddressProvider struct {
	mock.Mock
}

// GetRewardsAddress provides a mock function with given fields: context
func (_m *RewardsAddressProvider) GetRewardsAddress(context types.Context) (types.AccAddress, error) {
	ret := _m.Called(context)

	var r0 types.AccAddress
	var r1 error
	if rf, ok := ret.Get(0).(func(types.Context) (types.AccAddress, error)); ok {
		return rf(context)
	}
	if rf, ok := ret.Get(0).(func(types.Context) types.AccAddress); ok {
		r0 = rf(context)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.AccAddress)
		}
	}

	if rf, ok := ret.Get(1).(func(types.Context) error); ok {
		r1 = rf(context)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewRewardsAddressProvider creates a new instance of RewardsAddressProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewRewardsAddressProvider(t interface {
	mock.TestingT
	Cleanup(func())
}) *RewardsAddressProvider {
	mock := &RewardsAddressProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
