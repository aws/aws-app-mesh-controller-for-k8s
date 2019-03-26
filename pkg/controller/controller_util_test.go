package controller

import (
	"testing"
)

func TestConstructAppMeshVNodeNameFromCRD(t *testing.T) {
	t.Run("group", func(t *testing.T) {
		t.Run("noDot", func(t *testing.T) {
			originalName := "foo"
			actual := namespacedResourceName(originalName, "bar")
			expect := "foo-bar"
			if actual != expect {
				t.Errorf("got %v, expect %v", actual, expect)
			}
		})
		t.Run("oneDot", func(t *testing.T) {
			originalName := "foo.dummy"
			actual := namespacedResourceName(originalName, "bar")
			expect := "foo-dummy"
			if actual != expect {
				t.Errorf("got %v, expect %v", actual, expect)
			}
		})
		t.Run("twoDots", func(t *testing.T) {
			originalName := "foo.dummy.bummer"
			actual := namespacedResourceName(originalName, "bar")
			expect := "foo-dummy-bummer"
			if actual != expect {
				t.Errorf("got %v, expect %v", actual, expect)
			}
		})
	})
}
