package cloudmap

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

func Test_defaultInstancesHealthProber_filterInstancesBlockedByCMHealthyReadinessGate(t *testing.T) {
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-1",
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:   k8s.ConditionAWSCloudMapHealthy,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}
	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-2",
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:   k8s.ConditionAWSCloudMapHealthy,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	pod2New := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-2",
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:   k8s.ConditionAWSCloudMapHealthy,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}
	pod3 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-3",
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:   k8s.ConditionAWSCloudMapHealthy,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}
	pod3New := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-3",
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:   k8s.ConditionAWSCloudMapHealthy,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	type env struct {
		pods []*corev1.Pod
	}
	type args struct {
		instanceInfoByID map[string]instanceInfo
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    map[string]instanceInfo
		wantErr error
	}{
		{
			name: "should filter unready instances - when one of them unready",
			env:  env{pods: []*corev1.Pod{pod1, pod2}},
			args: args{
				instanceInfoByID: map[string]instanceInfo{
					"192.168.1.1": {
						pod: pod1.DeepCopy(),
					},
					"192.168.1.2": {
						pod: pod2.DeepCopy(),
					},
				},
			},
			want: map[string]instanceInfo{
				"192.168.1.1": {
					pod: pod1,
				},
			},
		},
		{
			name: "should filter unready instances - when latest pod state is unready",
			env:  env{pods: []*corev1.Pod{pod1, pod2New}},
			args: args{
				instanceInfoByID: map[string]instanceInfo{
					"192.168.1.1": {
						pod: pod1.DeepCopy(),
					},
					"192.168.1.2": {
						pod: pod2.DeepCopy(),
					},
				},
			},
			want: map[string]instanceInfo{
				"192.168.1.1": {
					pod: pod1,
				},
				"192.168.1.2": {
					pod: pod2New,
				},
			},
		},
		{
			name: "should filter unready instances - when both are unready",
			env:  env{pods: []*corev1.Pod{pod1, pod3}},
			args: args{
				instanceInfoByID: map[string]instanceInfo{
					"192.168.1.1": {
						pod: pod1.DeepCopy(),
					},
					"192.168.1.3": {
						pod: pod3.DeepCopy(),
					},
				},
			},
			want: map[string]instanceInfo{
				"192.168.1.1": {
					pod: pod1,
				},
				"192.168.1.3": {
					pod: pod3,
				},
			},
		},
		{
			name: "should filter unready instances - when latest pod state is ready",
			env:  env{pods: []*corev1.Pod{pod1, pod3New}},
			args: args{
				instanceInfoByID: map[string]instanceInfo{
					"192.168.1.1": {
						pod: pod1.DeepCopy(),
					},
					"192.168.1.3": {
						pod: pod3.DeepCopy(),
					},
				},
			},
			want: map[string]instanceInfo{
				"192.168.1.1": {
					pod: pod1,
				},
			},
		},
		{
			name: "should filter unready instances based on latest pod state",
			env:  env{pods: []*corev1.Pod{pod1, pod2New, pod3New}},
			args: args{
				instanceInfoByID: map[string]instanceInfo{
					"192.168.1.1": {
						pod: pod1.DeepCopy(),
					},
					"192.168.1.2": {
						pod: pod2.DeepCopy(),
					},
					"192.168.1.3": {
						pod: pod3.DeepCopy(),
					},
				},
			},
			want: map[string]instanceInfo{
				"192.168.1.1": {
					pod: pod1,
				},
				"192.168.1.2": {
					pod: pod2New,
				},
			},
		},
		{
			name: "should work for nil map",
			env:  env{},
			args: args{
				instanceInfoByID: nil,
			},
			want: map[string]instanceInfo{},
		},
		{
			name: "should work for empty map",
			env:  env{},
			args: args{
				instanceInfoByID: map[string]instanceInfo{},
			},
			want: map[string]instanceInfo{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			p := &defaultInstancesHealthProber{
				k8sClient: k8sClient,
			}
			for _, pod := range tt.env.pods {
				k8sClient.Create(ctx, pod.DeepCopy())
			}
			got, err := p.filterInstancesBlockedByCMHealthyReadinessGate(ctx, tt.args.instanceInfoByID)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opts := cmp.Options{
					equality.IgnoreFakeClientPopulatedFields(),
					cmp.AllowUnexported(instanceInfo{}),
				}
				assert.True(t, cmp.Equal(tt.want, got, opts), "diff", cmp.Diff(tt.want, got, opts))
			}
		})
	}
}

func Test_defaultInstancesHealthProber_updateInstanceProbeEntry(t *testing.T) {
	t10SecFromNow := -10 * time.Second
	t21SecFromNow := -21 * time.Second
	type env struct {
		pods []*corev1.Pod
	}
	type args struct {
		probeEntry                *instanceProbeEntry
		healthyStatus             bool
		lastTransitionTimeFromNow *time.Duration
	}
	tests := []struct {
		name           string
		env            env
		args           args
		wantProbeEntry *instanceProbeEntry
		wantContinue   bool
		wantErr        error
	}{
		{
			name: "should update lastHealthyStatus if lastTransitionTime is zero value - update to healthy",
			args: args{
				probeEntry: &instanceProbeEntry{
					instanceID: "192.168.1.1",
					instanceInfo: instanceInfo{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{
								Name: "my-pod",
							},
						},
					},
				},
				healthyStatus: true,
			},
			wantProbeEntry: &instanceProbeEntry{
				instanceID: "192.168.1.1",
				instanceInfo: instanceInfo{
					pod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "my-pod",
						},
					},
				},
				lastHealthyStatus: true,
			},
			wantContinue: true,
		},
		{
			name: "should update lastHealthyStatus if lastTransitionTime is zero value - update to unhealthy",
			args: args{
				probeEntry: &instanceProbeEntry{
					instanceID: "192.168.1.1",
					instanceInfo: instanceInfo{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{
								Name: "my-pod",
							},
						},
					},
				},
				healthyStatus: false,
			},
			wantProbeEntry: &instanceProbeEntry{
				instanceID: "192.168.1.1",
				instanceInfo: instanceInfo{
					pod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "my-pod",
						},
					},
				},
				lastHealthyStatus: false,
			},
			wantContinue: true,
		},
		{
			name: "should update lastHealthyStatus if lastTransitionTime is within 20 second - update to healthy",
			args: args{
				probeEntry: &instanceProbeEntry{
					instanceID: "192.168.1.1",
					instanceInfo: instanceInfo{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{
								Name: "my-pod",
							},
						},
					},
					lastHealthyStatus: false,
				},
				lastTransitionTimeFromNow: &t10SecFromNow,
				healthyStatus:             true,
			},
			wantProbeEntry: &instanceProbeEntry{
				instanceID: "192.168.1.1",
				instanceInfo: instanceInfo{
					pod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "my-pod",
						},
					},
				},
				lastHealthyStatus: true,
			},
			wantContinue: true,
		},
		{
			name: "should update lastHealthyStatus if lastTransitionTime is beyond 20 second - update to healthy",
			env: env{
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "my-pod",
						},
					},
				},
			},
			args: args{
				probeEntry: &instanceProbeEntry{
					instanceID: "192.168.1.1",
					instanceInfo: instanceInfo{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{
								Name: "my-pod",
							},
						},
					},
					lastHealthyStatus: false,
				},
				lastTransitionTimeFromNow: &t21SecFromNow,
				healthyStatus:             true,
			},
			wantProbeEntry: &instanceProbeEntry{
				instanceID: "192.168.1.1",
				instanceInfo: instanceInfo{
					pod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "my-pod",
						},
					},
				},
				lastHealthyStatus: true,
			},
			wantContinue: true,
		},
		{
			name: "should patch pod if lastTransitionTime is beyond 20 second - remains healthy",
			env: env{
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "my-pod",
						},
					},
				},
			},
			args: args{
				probeEntry: &instanceProbeEntry{
					instanceID: "192.168.1.1",
					instanceInfo: instanceInfo{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{
								Name: "my-pod",
							},
						},
					},
					lastHealthyStatus: true,
				},
				lastTransitionTimeFromNow: &t21SecFromNow,
				healthyStatus:             true,
			},
			wantProbeEntry: &instanceProbeEntry{
				instanceID: "192.168.1.1",
				instanceInfo: instanceInfo{
					pod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "my-pod",
						},
						Status: corev1.PodStatus{
							Conditions: []corev1.PodCondition{
								{
									Type:   k8s.ConditionAWSCloudMapHealthy,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
				lastHealthyStatus: true,
			},
			wantContinue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			p := &defaultInstancesHealthProber{
				k8sClient:          k8sClient,
				transitionDuration: defaultHealthTransitionDuration,
			}

			for _, pod := range tt.env.pods {
				k8sClient.Create(ctx, pod.DeepCopy())
			}
			if tt.args.lastTransitionTimeFromNow != nil {
				tt.args.probeEntry.lastTransitionTime = time.Now().Add(*tt.args.lastTransitionTimeFromNow)
			}

			gotContinue, gotErr := p.updateInstanceProbeEntry(ctx, tt.args.probeEntry, tt.args.healthyStatus)
			if tt.wantErr != nil {
				assert.EqualError(t, gotErr, tt.wantErr.Error())
			} else {
				assert.NoError(t, gotErr)
				assert.Equal(t, tt.wantContinue, gotContinue)
				opts := cmp.Options{
					equality.IgnoreFakeClientPopulatedFields(),
					cmp.AllowUnexported(instanceProbeEntry{}),
					cmp.AllowUnexported(instanceInfo{}),
					cmpopts.IgnoreFields(corev1.PodCondition{}, "LastProbeTime", "LastTransitionTime"),
				}
				tt.wantProbeEntry.lastTransitionTime = tt.args.probeEntry.lastTransitionTime
				assert.True(t, cmp.Equal(tt.wantProbeEntry, tt.args.probeEntry, opts),
					"diff", cmp.Diff(tt.wantProbeEntry, tt.args.probeEntry, opts))
			}
		})
	}
}
