// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"
	rest "k8s.io/client-go/rest"

	v1beta1 "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/clientset/versioned/typed/appmesh/v1beta1"
)

// AppmeshV1beta1Interface is an autogenerated mock type for the AppmeshV1beta1Interface type
type AppmeshV1beta1Interface struct {
	mock.Mock
}

// Meshes provides a mock function with given fields:
func (_m *AppmeshV1beta1Interface) Meshes() v1beta1.MeshInterface {
	ret := _m.Called()

	var r0 v1beta1.MeshInterface
	if rf, ok := ret.Get(0).(func() v1beta1.MeshInterface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1beta1.MeshInterface)
		}
	}

	return r0
}

// RESTClient provides a mock function with given fields:
func (_m *AppmeshV1beta1Interface) RESTClient() rest.Interface {
	ret := _m.Called()

	var r0 rest.Interface
	if rf, ok := ret.Get(0).(func() rest.Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(rest.Interface)
		}
	}

	return r0
}

// VirtualNodes provides a mock function with given fields: namespace
func (_m *AppmeshV1beta1Interface) VirtualNodes(namespace string) v1beta1.VirtualNodeInterface {
	ret := _m.Called(namespace)

	var r0 v1beta1.VirtualNodeInterface
	if rf, ok := ret.Get(0).(func(string) v1beta1.VirtualNodeInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1beta1.VirtualNodeInterface)
		}
	}

	return r0
}

// VirtualServices provides a mock function with given fields: namespace
func (_m *AppmeshV1beta1Interface) VirtualServices(namespace string) v1beta1.VirtualServiceInterface {
	ret := _m.Called(namespace)

	var r0 v1beta1.VirtualServiceInterface
	if rf, ok := ret.Get(0).(func(string) v1beta1.VirtualServiceInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1beta1.VirtualServiceInterface)
		}
	}

	return r0
}
