// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/references/resolver.go

// Package mock_references is a generated GoMock package.
package mock_references

import (
	context "context"
	reflect "reflect"

	v1beta2 "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	gomock "github.com/golang/mock/gomock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MockResolver is a mock of Resolver interface.
type MockResolver struct {
	ctrl     *gomock.Controller
	recorder *MockResolverMockRecorder
}

// MockResolverMockRecorder is the mock recorder for MockResolver.
type MockResolverMockRecorder struct {
	mock *MockResolver
}

// NewMockResolver creates a new mock instance.
func NewMockResolver(ctrl *gomock.Controller) *MockResolver {
	mock := &MockResolver{ctrl: ctrl}
	mock.recorder = &MockResolverMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockResolver) EXPECT() *MockResolverMockRecorder {
	return m.recorder
}

// ResolveBackendGroupReference mocks base method.
func (m *MockResolver) ResolveBackendGroupReference(ctx context.Context, obj v1.Object, ref v1beta2.BackendGroupReference) (*v1beta2.BackendGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResolveBackendGroupReference", ctx, obj, ref)
	ret0, _ := ret[0].(*v1beta2.BackendGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResolveBackendGroupReference indicates an expected call of ResolveBackendGroupReference.
func (mr *MockResolverMockRecorder) ResolveBackendGroupReference(ctx, obj, ref interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResolveBackendGroupReference", reflect.TypeOf((*MockResolver)(nil).ResolveBackendGroupReference), ctx, obj, ref)
}

// ResolveMeshReference mocks base method.
func (m *MockResolver) ResolveMeshReference(ctx context.Context, ref v1beta2.MeshReference) (*v1beta2.Mesh, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResolveMeshReference", ctx, ref)
	ret0, _ := ret[0].(*v1beta2.Mesh)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResolveMeshReference indicates an expected call of ResolveMeshReference.
func (mr *MockResolverMockRecorder) ResolveMeshReference(ctx, ref interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResolveMeshReference", reflect.TypeOf((*MockResolver)(nil).ResolveMeshReference), ctx, ref)
}

// ResolveVirtualGatewayReference mocks base method.
func (m *MockResolver) ResolveVirtualGatewayReference(ctx context.Context, obj v1.Object, ref v1beta2.VirtualGatewayReference) (*v1beta2.VirtualGateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResolveVirtualGatewayReference", ctx, obj, ref)
	ret0, _ := ret[0].(*v1beta2.VirtualGateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResolveVirtualGatewayReference indicates an expected call of ResolveVirtualGatewayReference.
func (mr *MockResolverMockRecorder) ResolveVirtualGatewayReference(ctx, obj, ref interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResolveVirtualGatewayReference", reflect.TypeOf((*MockResolver)(nil).ResolveVirtualGatewayReference), ctx, obj, ref)
}

// ResolveVirtualNodeReference mocks base method.
func (m *MockResolver) ResolveVirtualNodeReference(ctx context.Context, obj v1.Object, ref v1beta2.VirtualNodeReference) (*v1beta2.VirtualNode, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResolveVirtualNodeReference", ctx, obj, ref)
	ret0, _ := ret[0].(*v1beta2.VirtualNode)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResolveVirtualNodeReference indicates an expected call of ResolveVirtualNodeReference.
func (mr *MockResolverMockRecorder) ResolveVirtualNodeReference(ctx, obj, ref interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResolveVirtualNodeReference", reflect.TypeOf((*MockResolver)(nil).ResolveVirtualNodeReference), ctx, obj, ref)
}

// ResolveVirtualRouterReference mocks base method.
func (m *MockResolver) ResolveVirtualRouterReference(ctx context.Context, obj v1.Object, ref v1beta2.VirtualRouterReference) (*v1beta2.VirtualRouter, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResolveVirtualRouterReference", ctx, obj, ref)
	ret0, _ := ret[0].(*v1beta2.VirtualRouter)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResolveVirtualRouterReference indicates an expected call of ResolveVirtualRouterReference.
func (mr *MockResolverMockRecorder) ResolveVirtualRouterReference(ctx, obj, ref interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResolveVirtualRouterReference", reflect.TypeOf((*MockResolver)(nil).ResolveVirtualRouterReference), ctx, obj, ref)
}

// ResolveVirtualServiceReference mocks base method.
func (m *MockResolver) ResolveVirtualServiceReference(ctx context.Context, obj v1.Object, ref v1beta2.VirtualServiceReference) (*v1beta2.VirtualService, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResolveVirtualServiceReference", ctx, obj, ref)
	ret0, _ := ret[0].(*v1beta2.VirtualService)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResolveVirtualServiceReference indicates an expected call of ResolveVirtualServiceReference.
func (mr *MockResolverMockRecorder) ResolveVirtualServiceReference(ctx, obj, ref interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResolveVirtualServiceReference", reflect.TypeOf((*MockResolver)(nil).ResolveVirtualServiceReference), ctx, obj, ref)
}
