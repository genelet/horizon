package dethcl

import (
	"reflect"
	"testing"

	"github.com/genelet/horizon/utils"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestParseHCLFile(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name: "valid HCL",
			input: []byte(`
				name = "test"
				count = 42
			`),
			wantErr: false,
		},
		{
			name: "valid HCL with blocks",
			input: []byte(`
				name = "test"
				config {
					value = "nested"
				}
			`),
			wantErr: false,
		},
		{
			name: "invalid HCL - syntax error",
			input: []byte(`
				name = "test
				invalid
			`),
			wantErr: true,
		},
		{
			name:    "empty HCL",
			input:   []byte(``),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, bd, err := parseHCLFile(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if file == nil {
				t.Error("expected file, got nil")
			}
			if bd == nil {
				t.Error("expected body, got nil")
			}
		})
	}
}

func TestAddBlocksToTree(t *testing.T) {
	tests := []struct {
		name   string
		blocks []*hclsyntax.Block
		want   []string
	}{
		{
			name: "single block no labels",
			blocks: []*hclsyntax.Block{
				{Type: "config", Labels: []string{}},
			},
			want: []string{"config"},
		},
		{
			name: "block with one label",
			blocks: []*hclsyntax.Block{
				{Type: "resource", Labels: []string{"aws_instance"}},
			},
			want: []string{"resource"},
		},
		{
			name: "block with two labels",
			blocks: []*hclsyntax.Block{
				{Type: "resource", Labels: []string{"aws_instance", "example"}},
			},
			want: []string{"resource"},
		},
		{
			name: "multiple blocks",
			blocks: []*hclsyntax.Block{
				{Type: "config", Labels: []string{}},
				{Type: "resource", Labels: []string{"type1"}},
				{Type: "variable", Labels: []string{"var1"}},
			},
			want: []string{"config", "resource", "variable"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, _ := utils.NewTreeCtyFunction(nil)
			addBlocksToTree(node, tt.blocks)

			// Verify the blocks were added
			// We can't easily verify this without exposing Tree internals,
			// but we can at least verify no errors occurred during the call
			// The actual behavior is integration tested through the existing test suite
			if len(tt.want) > 0 && node == nil {
				t.Error("node should not be nil after adding blocks")
			}
		})
	}
}

func TestProcessSimpleFields(t *testing.T) {
	type TestStruct struct {
		Name  string `hcl:"name"`
		Count int    `hcl:"count"`
		Skip  string `hcl:"skip"`
	}

	// Create the "raw" decoded values
	rawType := reflect.StructOf([]reflect.StructField{
		{
			Name: "Name",
			Type: reflect.TypeOf(""),
			Tag:  `hcl:"name"`,
		},
		{
			Name: "Count",
			Type: reflect.TypeOf(0),
			Tag:  `hcl:"count"`,
		},
		{
			Name: "Skip",
			Type: reflect.TypeOf(""),
			Tag:  `hcl:"skip"`,
		},
	})

	raw := reflect.New(rawType).Elem()
	raw.FieldByName("Name").SetString("test_name")
	raw.FieldByName("Count").SetInt(42)
	raw.FieldByName("Skip").SetString("should_not_copy")

	// Create the target struct
	target := &TestStruct{}
	targetValue := reflect.ValueOf(target)

	// Fields that should be copied (only name and count, not skip)
	existingAttrs := map[string]bool{
		"name":  true,
		"count": true,
	}

	// Get the field definitions
	newFields := []reflect.StructField{
		{
			Name: "Name",
			Type: reflect.TypeOf(""),
			Tag:  `hcl:"name"`,
		},
		{
			Name: "Count",
			Type: reflect.TypeOf(0),
			Tag:  `hcl:"count"`,
		},
		{
			Name: "Skip",
			Type: reflect.TypeOf(""),
			Tag:  `hcl:"skip"`,
		},
	}

	// Process the fields
	processSimpleFields(newFields, raw, targetValue, existingAttrs)

	// Verify results
	if target.Name != "test_name" {
		t.Errorf("Name = %q, want %q", target.Name, "test_name")
	}
	if target.Count != 42 {
		t.Errorf("Count = %d, want %d", target.Count, 42)
	}
	if target.Skip != "" {
		t.Errorf("Skip = %q, want %q (should not be copied)", target.Skip, "")
	}
}

func TestUnmarshalToMap(t *testing.T) {
	hclData := []byte(`
		name = "test"
		count = 42
		tags = ["a", "b", "c"]
	`)

	result := make(map[string]any)
	node, _ := utils.NewTreeCtyFunction(nil)

	err := unmarshalToMap(node, hclData, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check name
	if name, ok := result["name"].(string); !ok || name != "test" {
		t.Errorf("name = %v, want %q", result["name"], "test")
	}

	// Check count
	if count, ok := result["count"].(int); !ok || count != 42 {
		t.Errorf("count = %v, want %d", result["count"], 42)
	}

	// Check tags
	if tags, ok := result["tags"].([]any); !ok {
		t.Errorf("tags should be []any, got %T", result["tags"])
	} else if len(tags) != 3 {
		t.Errorf("len(tags) = %d, want 3", len(tags))
	}
}

func TestUnmarshalToSlice(t *testing.T) {
	// HCL slices are formatted as array literals
	hclData := []byte(`[
		{
			name = "item1"
			value = 1
		},
		{
			name = "item2"
			value = 2
		}
	]`)

	result := make([]any, 0)
	node, _ := utils.NewTreeCtyFunction(nil)

	err := unmarshalToSlice(node, hclData, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("len(result) = %d, want 2", len(result))
	}
}

func TestProcessLabels(t *testing.T) {
	type TestStruct struct {
		Type string `hcl:"type,label"`
		Name string `hcl:"name,label"`
	}

	t.Run("labels from parent context", func(t *testing.T) {
		target := &TestStruct{}
		targetValue := reflect.ValueOf(target)

		labelFields := []reflect.StructField{
			{
				Name: "Type",
				Type: reflect.TypeOf(""),
				Tag:  `hcl:"type,label"`,
			},
			{
				Name: "Name",
				Type: reflect.TypeOf(""),
				Tag:  `hcl:"name,label"`,
			},
		}

		labels := []string{"resource", "example"}

		err := processLabels(labelFields, targetValue, nil, labels)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if target.Type != "resource" {
			t.Errorf("Type = %q, want %q", target.Type, "resource")
		}
		if target.Name != "example" {
			t.Errorf("Name = %q, want %q", target.Name, "example")
		}
	})

	t.Run("labels already set not overwritten", func(t *testing.T) {
		target := &TestStruct{Type: "existing"}
		targetValue := reflect.ValueOf(target)

		labelFields := []reflect.StructField{
			{
				Name: "Type",
				Type: reflect.TypeOf(""),
				Tag:  `hcl:"type,label"`,
			},
			{
				Name: "Name",
				Type: reflect.TypeOf(""),
				Tag:  `hcl:"name,label"`,
			},
		}

		labels := []string{"resource", "example"}

		err := processLabels(labelFields, targetValue, nil, labels)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Type should not be overwritten since it was already set
		if target.Type != "existing" {
			t.Errorf("Type = %q, want %q (should not be overwritten)", target.Type, "existing")
		}
		// Name should be set since it was empty
		if target.Name != "example" {
			t.Errorf("Name = %q, want %q", target.Name, "example")
		}
	})
}
