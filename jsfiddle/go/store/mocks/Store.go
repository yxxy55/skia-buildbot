// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"

	testing "testing"
)

// Store is an autogenerated mock type for the Store type
type Store struct {
	mock.Mock
}

// GetCode provides a mock function with given fields: hash, fiddleType
func (_m *Store) GetCode(hash string, fiddleType string) (string, error) {
	ret := _m.Called(hash, fiddleType)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(hash, fiddleType)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(hash, fiddleType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PutCode provides a mock function with given fields: code, fiddleType
func (_m *Store) PutCode(code string, fiddleType string) (string, error) {
	ret := _m.Called(code, fiddleType)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(code, fiddleType)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(code, fiddleType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewStore creates a new instance of Store. It also registers a cleanup function to assert the mocks expectations.
func NewStore(t testing.TB) *Store {
	mock := &Store{}

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
