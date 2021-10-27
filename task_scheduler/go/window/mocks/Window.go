// Code generated by mockery v2.4.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	repograph "go.skia.org/infra/go/git/repograph"

	time "time"
)

// Window is an autogenerated mock type for the Window type
type Window struct {
	mock.Mock
}

// EarliestStart provides a mock function with given fields:
func (_m *Window) EarliestStart() time.Time {
	ret := _m.Called()

	var r0 time.Time
	if rf, ok := ret.Get(0).(func() time.Time); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	return r0
}

// Start provides a mock function with given fields: repo
func (_m *Window) Start(repo string) time.Time {
	ret := _m.Called(repo)

	var r0 time.Time
	if rf, ok := ret.Get(0).(func(string) time.Time); ok {
		r0 = rf(repo)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	return r0
}

// StartTimesByRepo provides a mock function with given fields:
func (_m *Window) StartTimesByRepo() map[string]time.Time {
	ret := _m.Called()

	var r0 map[string]time.Time
	if rf, ok := ret.Get(0).(func() map[string]time.Time); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]time.Time)
		}
	}

	return r0
}

// TestCommit provides a mock function with given fields: repo, c
func (_m *Window) TestCommit(repo string, c *repograph.Commit) bool {
	ret := _m.Called(repo, c)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string, *repograph.Commit) bool); ok {
		r0 = rf(repo, c)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// TestCommitHash provides a mock function with given fields: repo, revision
func (_m *Window) TestCommitHash(repo string, revision string) (bool, error) {
	ret := _m.Called(repo, revision)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string, string) bool); ok {
		r0 = rf(repo, revision)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(repo, revision)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TestTime provides a mock function with given fields: repo, t
func (_m *Window) TestTime(repo string, t time.Time) bool {
	ret := _m.Called(repo, t)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string, time.Time) bool); ok {
		r0 = rf(repo, t)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Update provides a mock function with given fields: ctx
func (_m *Window) Update(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateWithTime provides a mock function with given fields: now
func (_m *Window) UpdateWithTime(now time.Time) error {
	ret := _m.Called(now)

	var r0 error
	if rf, ok := ret.Get(0).(func(time.Time) error); ok {
		r0 = rf(now)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
