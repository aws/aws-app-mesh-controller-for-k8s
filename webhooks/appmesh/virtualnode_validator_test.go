package appmesh

import (
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_virtualNodeValidator_enforceFieldsImmutability(t *testing.T) {
	type args struct {
		vn    *appmesh.VirtualNode
		oldVN *appmesh.VirtualNode
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "VirtualNode immutable fields didn't change",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "cloudmap-ns",
								ServiceName:   "cloudmap-svc",
							},
						},
					},
				},
				oldVN: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "cloudmap-ns",
								ServiceName:   "cloudmap-svc",
							},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "VirtualNode Optional ServiceDiscovery scenario",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
				oldVN: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "VirtualNode DNS Servicediscovery change is allowed",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							DNS: &appmesh.DNSServiceDiscovery{Hostname: "dns-new-hostname"},
						},
					},
				},
				oldVN: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							DNS: &appmesh.DNSServiceDiscovery{Hostname: "dns-hostname"},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "VirtualNode Servicediscovery mode change from DNS to CloudMap is allowed",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "cloudmap-ns",
								ServiceName:   "cloudmap-svc",
							},
						},
					},
				},
				oldVN: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							DNS: &appmesh.DNSServiceDiscovery{Hostname: "dns-hostname"},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "VirtualNode field awsName changed",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns_my-cluster"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "cloudmap-ns",
								ServiceName:   "cloudmap-svc",
							},
						},
					},
				},
				oldVN: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "cloudmap-ns",
								ServiceName:   "cloudmap-svc",
							},
						},
					},
				},
			},
			wantErr: errors.New("VirtualNode update may not change these fields: spec.awsName"),
		},
		{
			name: "VirtualNode field meshRef changed",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "cloudmap-ns",
								ServiceName:   "cloudmap-svc",
							},
						},
					},
				},
				oldVN: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "cloudmap-ns",
								ServiceName:   "cloudmap-svc",
							},
						},
					},
				},
			},
			wantErr: errors.New("VirtualNode update may not change these fields: spec.meshRef"),
		},
		{
			name: "VirtualNode field awsCloudMap changed",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "new-cloudmap-ns",
								ServiceName:   "new-cloudmap-svc",
							},
						},
					},
				},
				oldVN: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "cloudmap-ns",
								ServiceName:   "cloudmap-svc",
							},
						},
					},
				},
			},
			wantErr: errors.New("VirtualNode update may not change these fields: spec.serviceDiscovery.awsCloudMap"),
		},
		{
			name: "VirtualNode fields awsName, meshRef and awsCloudMap changed",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns-my-cluster"),
						MeshRef: &appmesh.MeshReference{
							Name: "another-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "new-cloudmap-ns",
								ServiceName:   "new-cloudmap-svc",
							},
						},
					},
				},
				oldVN: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "cloudmap-ns",
								ServiceName:   "cloudmap-svc",
							},
						},
					},
				},
			},
			wantErr: errors.New("VirtualNode update may not change these fields: spec.awsName,spec.meshRef,spec.serviceDiscovery.awsCloudMap"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &virtualNodeValidator{}
			err := v.enforceFieldsImmutability(tt.args.vn, tt.args.oldVN)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_virtualNodeValidator_checkVirtualNodeBackendsForDuplicates(t *testing.T) {
	testARN := "testARN"
	testPrimaryNamespace := "awesome-ns"
	testSecondaryNamespace := "secondary-ns"
	type args struct {
		vn    *appmesh.VirtualNode
		oldVN *appmesh.VirtualNode
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "Duplicate VirtualService Reference (By Name) included",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "cloudmap-ns",
								ServiceName:   "cloudmap-svc",
							},
						},
						Backends: []appmesh.Backend{
							{VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceRef: &appmesh.VirtualServiceReference{
									Name:      "testVS",
									Namespace: &testPrimaryNamespace,
								},
							}},
							{VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceRef: &appmesh.VirtualServiceReference{
									Name:      "testVS",
									Namespace: &testPrimaryNamespace,
								},
							},
							},
						},
					},
				},
			},
			wantErr: errors.New("VirtualNode-my-vn has duplicate VirtualServiceReferences testVS"),
		},
		{
			name: "Duplicate VirtualService ARN included",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "cloudmap-ns",
								ServiceName:   "cloudmap-svc",
							},
						},
						Backends: []appmesh.Backend{
							{VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceARN: &testARN,
							},
							},
							{VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceARN: &testARN,
							},
							},
						},
					},
				},
			},
			wantErr: errors.New("VirtualNode-my-vn has duplicate VirtualServiceReferenceARNs testARN"),
		},
		{
			name: "No Duplicate Backends",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "cloudmap-ns",
								ServiceName:   "cloudmap-svc",
							},
						},
						Backends: []appmesh.Backend{
							{VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceRef: &appmesh.VirtualServiceReference{
									Name:      "testVS",
									Namespace: &testPrimaryNamespace,
								},
							}},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "Same VirtualService name across different namespaces",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						ServiceDiscovery: &appmesh.ServiceDiscovery{
							AWSCloudMap: &appmesh.AWSCloudMapServiceDiscovery{
								NamespaceName: "cloudmap-ns",
								ServiceName:   "cloudmap-svc",
							},
						},
						Backends: []appmesh.Backend{
							{VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceRef: &appmesh.VirtualServiceReference{
									Name:      "testVS",
									Namespace: &testPrimaryNamespace,
								},
							}},
							{VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceRef: &appmesh.VirtualServiceReference{
									Name:      "testVS",
									Namespace: &testSecondaryNamespace,
								},
							}},
						},
					},
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &virtualNodeValidator{}
			err := v.checkVirtualNodeBackendsForDuplicates(tt.args.vn)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_virtualNodeValidator_checkRequiredFields(t *testing.T) {
	testARN := "testARN"
	type args struct {
		vn    *appmesh.VirtualNode
		oldVN *appmesh.VirtualNode
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "ServiceDiscovery is Optional if listeners are not specified",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "my-vn",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						Backends: []appmesh.Backend{
							{VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceARN: &testARN,
							},
							},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "ServiceDiscovery is Mandatory if listeners are specified",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "awesome-ns",
						Name:      "TestVN",
					},
					Spec: appmesh.VirtualNodeSpec{
						AWSName: aws.String("my-vn_awesome-ns"),
						MeshRef: &appmesh.MeshReference{
							Name: "my-mesh",
							UID:  "408d3036-7dec-11ea-b156-0e30aabe1ca8",
						},
						Listeners: []appmesh.Listener{
							{
								PortMapping: appmesh.PortMapping{
									Port:     8080,
									Protocol: "http",
								},
							},
						},
						Backends: []appmesh.Backend{
							{VirtualService: appmesh.VirtualServiceBackend{
								VirtualServiceARN: &testARN,
							},
							},
						},
					},
				},
			},
			wantErr: errors.New("ServiceDiscovery missing for VirtualNode-TestVN. ServiceDiscovery must be specified when a listener is specified."),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &virtualNodeValidator{}
			err := v.checkForRequiredFields(tt.args.vn)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
