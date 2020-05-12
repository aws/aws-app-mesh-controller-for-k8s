package references

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

func Test_buildIndexKey(t *testing.T) {
	type args struct {
		referentKind string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal case",
			args: args{
				referentKind: "VirtualNode",
			},
			want: "objectRefIndex:VirtualNode",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildIndexKey(tt.args.referentKind)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_buildIndexValue(t *testing.T) {
	type args struct {
		referentKey types.NamespacedName
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal case",
			args: args{
				referentKey: types.NamespacedName{
					Namespace: "my-ns",
					Name:      "my-obj",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildIndexValue(tt.args.referentKey)
			assert.Equal(t, tt.want, got)
		})
	}
}
