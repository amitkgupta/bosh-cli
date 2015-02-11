// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/cloudfoundry/bosh-micro-cli/installation/pkg (interfaces: Installer)

package mocks

import (
	gomock "code.google.com/p/gomock/gomock"
	pkg "github.com/cloudfoundry/bosh-micro-cli/installation/pkg"
)

// Mock of Installer interface
type MockInstaller struct {
	ctrl     *gomock.Controller
	recorder *_MockInstallerRecorder
}

// Recorder for MockInstaller (not exported)
type _MockInstallerRecorder struct {
	mock *MockInstaller
}

func NewMockInstaller(ctrl *gomock.Controller) *MockInstaller {
	mock := &MockInstaller{ctrl: ctrl}
	mock.recorder = &_MockInstallerRecorder{mock}
	return mock
}

func (_m *MockInstaller) EXPECT() *_MockInstallerRecorder {
	return _m.recorder
}

func (_m *MockInstaller) Install(_param0 pkg.CompiledPackageRef, _param1 string) error {
	ret := _m.ctrl.Call(_m, "Install", _param0, _param1)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockInstallerRecorder) Install(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Install", arg0, arg1)
}