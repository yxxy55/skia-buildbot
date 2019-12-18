// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	context "context"

	code_review "go.skia.org/infra/golden/go/code_review"

	mock "github.com/stretchr/testify/mock"

	vcsinfo "go.skia.org/infra/go/vcsinfo"
)

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

// CommentOn provides a mock function with given fields: ctx, clID, message
func (_m *Client) CommentOn(ctx context.Context, clID string, message string) error {
	ret := _m.Called(ctx, clID, message)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, clID, message)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetChangeList provides a mock function with given fields: ctx, id
func (_m *Client) GetChangeList(ctx context.Context, id string) (code_review.ChangeList, error) {
	ret := _m.Called(ctx, id)

	var r0 code_review.ChangeList
	if rf, ok := ret.Get(0).(func(context.Context, string) code_review.ChangeList); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Get(0).(code_review.ChangeList)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetChangeListIDForCommit provides a mock function with given fields: ctx, commit
func (_m *Client) GetChangeListIDForCommit(ctx context.Context, commit *vcsinfo.LongCommit) (string, error) {
	ret := _m.Called(ctx, commit)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, *vcsinfo.LongCommit) string); ok {
		r0 = rf(ctx, commit)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *vcsinfo.LongCommit) error); ok {
		r1 = rf(ctx, commit)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPatchSets provides a mock function with given fields: ctx, clID
func (_m *Client) GetPatchSets(ctx context.Context, clID string) ([]code_review.PatchSet, error) {
	ret := _m.Called(ctx, clID)

	var r0 []code_review.PatchSet
	if rf, ok := ret.Get(0).(func(context.Context, string) []code_review.PatchSet); ok {
		r0 = rf(ctx, clID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]code_review.PatchSet)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, clID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// System provides a mock function with given fields:
func (_m *Client) System() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
