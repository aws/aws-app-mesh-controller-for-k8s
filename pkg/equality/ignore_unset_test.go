package equality

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"testing"
)

type NestedStruct struct {
	StrField string
}

type TestStruct struct {
	StrField       string
	SliceField     []string
	MapField       map[string]string
	StructField    NestedStruct
	StructPtrField *NestedStruct
}

func TestIgnoreLeftHandUnset(t *testing.T) {
	tests := []struct {
		name       string
		argLeft    TestStruct
		argRight   TestStruct
		wantEquals bool
	}{
		{
			name: "when StrField equals",
			argLeft: TestStruct{
				StrField: "a",
			},
			argRight: TestStruct{
				StrField: "a",
			},
			wantEquals: true,
		},
		{
			name: "when StrField differs",
			argLeft: TestStruct{
				StrField: "a",
			},
			argRight: TestStruct{
				StrField: "b",
			},
			wantEquals: false,
		},
		{
			name: "when SliceField equals",
			argLeft: TestStruct{
				SliceField: []string{"a", "b"},
			},
			argRight: TestStruct{
				SliceField: []string{"a", "b"},
			},
			wantEquals: true,
		},
		{
			name: "when SliceField differs",
			argLeft: TestStruct{
				SliceField: []string{"a", "b"},
			},
			argRight: TestStruct{
				SliceField: []string{"b", "a"},
			},
			wantEquals: false,
		},
		{
			name: "when left hand arg have nil SliceField",
			argLeft: TestStruct{
				SliceField: nil,
			},
			argRight: TestStruct{
				SliceField: []string{"b", "a"},
			},
			wantEquals: true,
		},
		{
			name: "when left hand arg have non-nil but empty SliceField",
			argLeft: TestStruct{
				SliceField: []string{},
			},
			argRight: TestStruct{
				SliceField: []string{"b", "a"},
			},
			wantEquals: false,
		},
		{
			name: "when MapField equals",
			argLeft: TestStruct{
				MapField: map[string]string{"k": "v"},
			},
			argRight: TestStruct{
				MapField: map[string]string{"k": "v"},
			},
			wantEquals: true,
		},
		{
			name: "when MapField differs by value",
			argLeft: TestStruct{
				MapField: map[string]string{"k": "v1"},
			},
			argRight: TestStruct{
				MapField: map[string]string{"k": "v2"},
			},
			wantEquals: false,
		},
		{
			name: "when MapField differs by key",
			argLeft: TestStruct{
				MapField: map[string]string{"k1": "v"},
			},
			argRight: TestStruct{
				MapField: map[string]string{"k2": "v"},
			},
			wantEquals: false,
		},
		{
			name: "when left hand arg have nil MapField",
			argLeft: TestStruct{
				MapField: nil,
			},
			argRight: TestStruct{
				MapField: map[string]string{"k": "v"},
			},
			wantEquals: true,
		},
		{
			name: "when left hand arg have non-nil but empty MapField",
			argLeft: TestStruct{
				MapField: map[string]string{},
			},
			argRight: TestStruct{
				MapField: map[string]string{"k": "v"},
			},
			wantEquals: false,
		},
		{
			name: "when StructField equals",
			argLeft: TestStruct{
				StructField: NestedStruct{
					StrField: "a",
				},
			},
			argRight: TestStruct{
				StructField: NestedStruct{
					StrField: "a",
				},
			},
			wantEquals: true,
		},
		{
			name: "when StructField differs",
			argLeft: TestStruct{
				StructField: NestedStruct{
					StrField: "a",
				},
			},
			argRight: TestStruct{
				StructField: NestedStruct{
					StrField: "b",
				},
			},
			wantEquals: false,
		},
		{
			name: "when StructPtrField equals",
			argLeft: TestStruct{
				StructPtrField: &NestedStruct{
					StrField: "a",
				},
			},
			argRight: TestStruct{
				StructPtrField: &NestedStruct{
					StrField: "a",
				},
			},
			wantEquals: true,
		},
		{
			name: "when StructPtrField differs",
			argLeft: TestStruct{
				StructPtrField: &NestedStruct{
					StrField: "a",
				},
			},
			argRight: TestStruct{
				StructPtrField: &NestedStruct{
					StrField: "b",
				},
			},
			wantEquals: false,
		},
		{
			name: "when left hand arg have nil StructPtrField",
			argLeft: TestStruct{
				StructPtrField: nil,
			},
			argRight: TestStruct{
				StructPtrField: &NestedStruct{
					StrField: "b",
				},
			},
			wantEquals: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := IgnoreLeftHandUnset()
			gotEquals := cmp.Equal(tt.argLeft, tt.argRight, opts)
			assert.Equal(t, tt.wantEquals, gotEquals)
		})
	}
}
