package dethcl

import (
	"testing"

	"github.com/genelet/schema"
)

// Test Quick Start sample
func TestReadmeQuickStart(t *testing.T) {
	type Config struct {
		Name    string `hcl:"name"`
		Enabled bool   `hcl:"enabled,optional"`
	}

	hclData := []byte(`
		name = "example"
		enabled = true
	`)
	var cfg Config
	err := Unmarshal(hclData, &cfg)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if cfg.Name != "example" {
		t.Errorf("Expected name 'example', got '%s'", cfg.Name)
	}
	if !cfg.Enabled {
		t.Error("Expected enabled true")
	}

	// Marshal to HCL
	data, err := Marshal(&cfg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("Expected non-empty output")
	}
}

// Test 1.2 Encoding Map sample
func TestReadmeEncodingMap(t *testing.T) {
	type squareLocal struct {
		SX int `json:"sx" hcl:"sx"`
		SY int `json:"sy" hcl:"sy"`
	}

	type geometry struct {
		Name   string                  `json:"name" hcl:"name"`
		Shapes map[string]*squareLocal `json:"shapes" hcl:"shapes"`
	}

	app := &geometry{
		Name: "Medium Article",
		Shapes: map[string]*squareLocal{
			"k1": {SX: 2, SY: 3}, "k2": {SX: 5, SY: 6}},
	}

	bs, err := Marshal(app)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if len(bs) == 0 {
		t.Error("Expected non-empty output")
	}
}

// Test 1.3 Encode Interface Data sample
// Uses package-level types: inter, square, circle from sample_test.go
func TestReadmeEncodeInterface(t *testing.T) {
	type picture struct {
		Name     string  `json:"name" hcl:"name"`
		Drawings []inter `json:"drawings" hcl:"drawings"`
	}

	app := &picture{
		Name: "Medium Article",
		Drawings: []inter{
			&square{SX: 2, SY: 3}, &circle{Radius: 5.6}},
	}

	bs, err := Marshal(app)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if len(bs) == 0 {
		t.Error("Expected non-empty output")
	}
}

// Test 1.4 Encoding with HCL Labels sample
// Uses package-level types: inter, square, moresquare from sample_test.go
func TestReadmeEncodingLabels(t *testing.T) {
	type picture struct {
		Name     string  `json:"name" hcl:"name"`
		Drawings []inter `json:"drawings" hcl:"drawings"`
	}

	app := &picture{
		Name: "Medium Article",
		Drawings: []inter{
			&square{SX: 2, SY: 3},
			&moresquare{Morename1: "abc2", Morename2: "def2", SX: 2, SY: 3},
		},
	}

	bs, err := Marshal(app)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if len(bs) == 0 {
		t.Error("Expected non-empty output")
	}
}

// Types for the child unmarshal test (Chapter 2.4 example)
type readmeChild struct {
	Brand map[string]*toy `json:"brand" hcl:"brand,block"`
	Age   int             `json:"age" hcl:"age"`
}

// Test 2.4 Unmarshal HCL Data to Object (child example)
// Uses package-level types: inter, square, circle, geo, toy from sample_test.go
func TestReadmeUnmarshalChild(t *testing.T) {
	data1 := `
age = 5
brand "abc1" {
    toy_name = "roblox"
    price = 99.9
    geo {
        name = "medium shape"
        shape {
            radius = 1.234
        }
    }
}
brand "def2" {
    toy_name = "minecraft"
    price = 9.9
    geo {
        name = "square shape"
        shape {
            sx = 5
            sy = 6
        }
    }
}
`
	spec, err := schema.NewStruct("readmeChild", map[string]any{
		"Brand": map[string][2]any{
			"abc1": {"toy", map[string]any{
				"Geo": [2]any{
					"geo", map[string]any{"Shape": "circle"}}}},
			"def2": {"toy", map[string]any{
				"Geo": [2]any{
					"geo", map[string]any{"Shape": "square"}}}},
		},
	})
	if err != nil {
		t.Fatalf("NewStruct failed: %v", err)
	}
	ref := map[string]any{"toy": &toy{}, "geo": &geo{}, "circle": &circle{}, "square": &square{}, "readmeChild": &readmeChild{}}

	c := new(readmeChild)
	err = UnmarshalSpec([]byte(data1), c, spec, ref)
	if err != nil {
		t.Fatalf("UnmarshalSpec failed: %v", err)
	}
	if c.Age != 5 {
		t.Errorf("Expected Age 5, got %d", c.Age)
	}
	if c.Brand["abc1"] == nil {
		t.Error("Expected Brand abc1")
	}
	if c.Brand["abc1"].ToyName != "roblox" {
		t.Errorf("Expected ToyName 'roblox', got '%s'", c.Brand["abc1"].ToyName)
	}
	if c.Brand["def2"] == nil {
		t.Error("Expected Brand def2")
	}
	if c.Brand["def2"].ToyName != "minecraft" {
		t.Errorf("Expected ToyName 'minecraft', got '%s'", c.Brand["def2"].ToyName)
	}
}
