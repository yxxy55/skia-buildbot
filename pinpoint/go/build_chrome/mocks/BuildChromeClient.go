// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	context "context"

	buildbucketpb "go.chromium.org/luci/buildbucket/proto"

	mock "github.com/stretchr/testify/mock"

	swarming "go.chromium.org/luci/common/api/swarming/swarming/v1"
)

// BuildChromeClient is an autogenerated mock type for the BuildChromeClient type
type BuildChromeClient struct {
	mock.Mock
}

// CancelBuild provides a mock function with given fields: _a0, _a1, _a2
func (_m *BuildChromeClient) CancelBuild(_a0 context.Context, _a1 int64, _a2 string) error {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for CancelBuild")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64, string) error); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetStatus provides a mock function with given fields: _a0, _a1
func (_m *BuildChromeClient) GetStatus(_a0 context.Context, _a1 int64) (buildbucketpb.Status, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for GetStatus")
	}

	var r0 buildbucketpb.Status
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, int64) (buildbucketpb.Status, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int64) buildbucketpb.Status); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(buildbucketpb.Status)
	}

	if rf, ok := ret.Get(1).(func(context.Context, int64) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RetrieveCAS provides a mock function with given fields: _a0, _a1, _a2
func (_m *BuildChromeClient) RetrieveCAS(_a0 context.Context, _a1 int64, _a2 string) (*swarming.SwarmingRpcsCASReference, error) {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for RetrieveCAS")
	}

	var r0 *swarming.SwarmingRpcsCASReference
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, int64, string) (*swarming.SwarmingRpcsCASReference, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int64, string) *swarming.SwarmingRpcsCASReference); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swarming.SwarmingRpcsCASReference)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, int64, string) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SearchOrBuild provides a mock function with given fields: ctx, pinpointJobID, commit, device, deps, patches
func (_m *BuildChromeClient) SearchOrBuild(ctx context.Context, pinpointJobID string, commit string, device string, deps map[string]interface{}, patches []*buildbucketpb.GerritChange) (int64, error) {
	ret := _m.Called(ctx, pinpointJobID, commit, device, deps, patches)

	if len(ret) == 0 {
		panic("no return value specified for SearchOrBuild")
	}

	var r0 int64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, map[string]interface{}, []*buildbucketpb.GerritChange) (int64, error)); ok {
		return rf(ctx, pinpointJobID, commit, device, deps, patches)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, map[string]interface{}, []*buildbucketpb.GerritChange) int64); ok {
		r0 = rf(ctx, pinpointJobID, commit, device, deps, patches)
	} else {
		r0 = ret.Get(0).(int64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, map[string]interface{}, []*buildbucketpb.GerritChange) error); ok {
		r1 = rf(ctx, pinpointJobID, commit, device, deps, patches)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewBuildChromeClient creates a new instance of BuildChromeClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBuildChromeClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *BuildChromeClient {
	mock := &BuildChromeClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
