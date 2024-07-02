<<<<<<< HEAD
// Code generated by mockery v2.40.1. DO NOT EDIT.
=======
// Code generated by mockery v2.43.2. DO NOT EDIT.
>>>>>>> f1cde2a (fix: mempool lane size check on `CheckTx` (#561))

package mocks

import (
	context "context"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	mock "github.com/stretchr/testify/mock"

	types "github.com/cosmos/cosmos-sdk/types"
)

// StakingKeeper is an autogenerated mock type for the StakingKeeper type
type StakingKeeper struct {
	mock.Mock
}

// GetValidatorByConsAddr provides a mock function with given fields: _a0, _a1
func (_m *StakingKeeper) GetValidatorByConsAddr(_a0 context.Context, _a1 types.ConsAddress) (stakingtypes.Validator, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for GetValidatorByConsAddr")
	}

	var r0 stakingtypes.Validator
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, types.ConsAddress) (stakingtypes.Validator, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, types.ConsAddress) stakingtypes.Validator); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(stakingtypes.Validator)
	}

	if rf, ok := ret.Get(1).(func(context.Context, types.ConsAddress) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewStakingKeeper creates a new instance of StakingKeeper. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewStakingKeeper(t interface {
	mock.TestingT
	Cleanup(func())
},
) *StakingKeeper {
	mock := &StakingKeeper{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
