package dethcl

import (
	"testing"
)

// Test types for collectStructTypesFromObject
type testShape interface {
	Area() float64
}

type testCircle struct {
	Radius float64 `hcl:"radius"`
}

func (c *testCircle) Area() float64 {
	return 3.14159 * c.Radius * c.Radius
}

type testSquare struct {
	Side float64 `hcl:"side"`
}

func (s *testSquare) Area() float64 {
	return s.Side * s.Side
}

type testGeo struct {
	Name  string    `hcl:"name"`
	Shape testShape `hcl:"shape,block"`
}

type testDrawing struct {
	Title string  `hcl:"title"`
	Geo   testGeo `hcl:"geo,block"`
}

type testGallery struct {
	Name     string                `hcl:"name"`
	Drawings map[string]testGeo    `hcl:"drawings,block"`
	Items    []testGeo             `hcl:"items,block"`
}

func Test_collectStructTypesFromObject(t *testing.T) {
	t.Run("simple struct", func(t *testing.T) {
		type Simple struct {
			Name string `hcl:"name"`
		}
		ref := collectStructTypesFromObject(new(Simple), nil)
		if ref["Simple"] == nil {
			t.Error("expected Simple to be in ref")
		}
	})

	t.Run("nested struct", func(t *testing.T) {
		ref := collectStructTypesFromObject(new(testDrawing), nil)
		if ref["testDrawing"] == nil {
			t.Error("expected testDrawing to be in ref")
		}
		if ref["testGeo"] == nil {
			t.Error("expected testGeo to be in ref")
		}
	})

	t.Run("with interface implementations", func(t *testing.T) {
		implementations := map[string][]any{
			"testShape": {new(testCircle), new(testSquare)},
		}
		ref := collectStructTypesFromObject(new(testGeo), implementations)
		if ref["testGeo"] == nil {
			t.Error("expected testGeo to be in ref")
		}
		if ref["testCircle"] == nil {
			t.Error("expected testCircle to be in ref")
		}
		if ref["testSquare"] == nil {
			t.Error("expected testSquare to be in ref")
		}
	})

	t.Run("map field", func(t *testing.T) {
		ref := collectStructTypesFromObject(new(testGallery), nil)
		if ref["testGallery"] == nil {
			t.Error("expected testGallery to be in ref")
		}
		if ref["testGeo"] == nil {
			t.Error("expected testGeo to be in ref (from map value)")
		}
	})

	t.Run("slice field", func(t *testing.T) {
		ref := collectStructTypesFromObject(new(testGallery), nil)
		if ref["testGeo"] == nil {
			t.Error("expected testGeo to be in ref (from slice element)")
		}
	})

	t.Run("package qualified names", func(t *testing.T) {
		ref := collectStructTypesFromObject(new(testGeo), nil)
		// Should have both short and package-qualified name
		if ref["testGeo"] == nil {
			t.Error("expected testGeo short name to be in ref")
		}
		if ref["dethcl.testGeo"] == nil {
			t.Error("expected dethcl.testGeo package-qualified name to be in ref")
		}
	})

	t.Run("nil input", func(t *testing.T) {
		ref := collectStructTypesFromObject(nil, nil)
		if len(ref) != 0 {
			t.Error("expected empty ref for nil input")
		}
	})

	t.Run("non-struct input", func(t *testing.T) {
		str := "hello"
		ref := collectStructTypesFromObject(&str, nil)
		if len(ref) != 0 {
			t.Error("expected empty ref for non-struct input")
		}
	})
}
