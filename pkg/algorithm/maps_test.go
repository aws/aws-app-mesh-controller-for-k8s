package algorithm

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMergeStringMap(t *testing.T) {
	type args struct {
		maps []map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "merge two maps without duplicates",
			args: args{
				maps: []map[string]string{
					{
						"a": "1",
						"b": "2",
					},
					{
						"c": "3",
						"d": "4",
					},
				},
			},
			want: map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
				"d": "4",
			},
		},
		{
			name: "merge two maps with duplicates",
			args: args{
				maps: []map[string]string{
					{
						"a": "1",
						"b": "2",
					},
					{
						"a": "3",
						"d": "4",
					},
				},
			},
			want: map[string]string{
				"a": "1",
				"b": "2",
				"d": "4",
			},
		},
		{
			name: "merge two maps when first map is nil",
			args: args{
				maps: []map[string]string{
					nil,
					{
						"c": "3",
						"d": "4",
					},
				},
			},
			want: map[string]string{
				"c": "3",
				"d": "4",
			},
		},
		{
			name: "merge two maps when second map is nil",
			args: args{
				maps: []map[string]string{
					{
						"a": "1",
						"b": "2",
					},
					nil,
				},
			},
			want: map[string]string{
				"a": "1",
				"b": "2",
			},
		},
		{
			name: "merge two maps when both map is nil",
			args: args{
				maps: []map[string]string{
					nil,
					nil,
				},
			},
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeStringMap(tt.args.maps...)
			assert.Equal(t, tt.want, got)
		})
	}
}
