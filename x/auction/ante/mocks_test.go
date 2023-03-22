package ante_test

import (
	"reflect"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/golang/mock/gomock"
)

type MockAccountKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockAccountKeeperMockRecorder
}

type MockAccountKeeperMockRecorder struct {
	mock *MockAccountKeeper
}

func NewMockAccountKeeper(ctrl *gomock.Controller) *MockAccountKeeper {
	mock := &MockAccountKeeper{ctrl: ctrl}
	mock.recorder = &MockAccountKeeperMockRecorder{mock}
	return mock
}

func (m *MockAccountKeeper) EXPECT() *MockAccountKeeperMockRecorder {
	return m.recorder
}

func (m *MockAccountKeeper) GetModuleAddress(name string) sdk.AccAddress {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModuleAddress", name)
	ret0, _ := ret[0].(sdk.AccAddress)
	return ret0
}

func (mr *MockAccountKeeperMockRecorder) GetModuleAddress(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModuleAddress", reflect.TypeOf((*MockAccountKeeper)(nil).GetModuleAddress), name)
}

type MockBankKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockBankKeeperMockRecorder
}

type MockBankKeeperMockRecorder struct {
	mock *MockBankKeeper
}

func NewMockBankKeeper(ctrl *gomock.Controller) *MockBankKeeper {
	mock := &MockBankKeeper{ctrl: ctrl}
	mock.recorder = &MockBankKeeperMockRecorder{mock}
	return mock
}

func (m *MockBankKeeper) EXPECT() *MockBankKeeperMockRecorder {
	return m.recorder
}

func (m *MockBankKeeper) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllBalances", ctx, addr)
	ret0 := ret[0].(sdk.Coins)
	return ret0
}

func (mr *MockBankKeeperMockRecorder) GetAllBalances(ctx, addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllBalances", reflect.TypeOf((*MockBankKeeper)(nil).GetAllBalances), ctx, addr)
}

func (m *MockBankKeeper) SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SendCoins", ctx, fromAddr, toAddr, amt)
	return nil
}

func (mr *MockBankKeeperMockRecorder) SendCoins(ctx, fromAddr, toAddr, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendCoins", reflect.TypeOf((*MockBankKeeper)(nil).SendCoins), ctx, fromAddr, toAddr, amt)
}

type MockDistributionKeeperRecorder struct {
	mock *MockDistributionKeeper
}

type MockDistributionKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockDistributionKeeperRecorder
}

func NewMockDistributionKeeper(ctrl *gomock.Controller) *MockDistributionKeeper {
	mock := &MockDistributionKeeper{ctrl: ctrl}
	mock.recorder = &MockDistributionKeeperRecorder{mock}
	return mock
}

func (m *MockDistributionKeeper) EXPECT() *MockDistributionKeeperRecorder {
	return m.recorder
}

func (m *MockDistributionKeeper) GetPreviousProposerConsAddr(ctx sdk.Context) sdk.ConsAddress {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPreviousProposerConsAddr", ctx)
	ret0 := ret[0].(sdk.ConsAddress)
	return ret0
}

func (mr *MockDistributionKeeperRecorder) GetPreviousProposerConsAddr(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPreviousProposerConsAddr", reflect.TypeOf((*MockDistributionKeeper)(nil).GetPreviousProposerConsAddr), ctx)
}

type MockStakingKeeperRecorder struct {
	mock *MockStakingKeeper
}

type MockStakingKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockStakingKeeperRecorder
}

func NewMockStakingKeeper(ctrl *gomock.Controller) *MockStakingKeeper {
	mock := &MockStakingKeeper{ctrl: ctrl}
	mock.recorder = &MockStakingKeeperRecorder{mock}
	return mock
}

func (m *MockStakingKeeper) EXPECT() *MockStakingKeeperRecorder {
	return m.recorder
}

func (m *MockStakingKeeper) ValidatorByConsAddr(ctx sdk.Context, consAddr sdk.ConsAddress) stakingtypes.ValidatorI {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidatorByConsAddr", ctx, consAddr)
	ret0 := ret[0].(stakingtypes.ValidatorI)
	return ret0
}

func (mr *MockStakingKeeperRecorder) ValidatorByConsAddr(ctx, consAddr any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidatorByConsAddr", reflect.TypeOf((*MockStakingKeeper)(nil).ValidatorByConsAddr), ctx, consAddr)
}
