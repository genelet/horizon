package convert

import (
	"testing"
)

// Test Chapter 3.3 HCL to YAML conversion sample
func TestReadmeHCLToYAML(t *testing.T) {
	bs := []byte(`parties = [
  "one",
  "two",
  [
    "three",
    "four"
  ],
  {
    five = "51"
    six = 61
  }
]
roads {
  y = "b"
  z {
    za = "aa"
    zb = 3.14
  }
  x = "a"
  xy = [
    "ab",
    true
  ]
}
name = "marcus"
num = 2
radius = 1
`)
	yml, err := HCLToYAML(bs)
	if err != nil {
		t.Fatalf("HCLToYAML failed: %v", err)
	}
	if len(yml) == 0 {
		t.Error("Expected non-empty output")
	}
	// Check some key fields are present
	output := string(yml)
	if !containsString(output, "name: marcus") {
		t.Errorf("Expected 'name: marcus' in output, got:\n%s", output)
	}
	if !containsString(output, "num: 2") {
		t.Errorf("Expected 'num: 2' in output, got:\n%s", output)
	}
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test HCL to JSON conversion
func TestReadmeHCLToJSON(t *testing.T) {
	hcl := []byte(`
version = "3.7"
services "db" {
  image = "hashicorpdemoapp/product-api-db:v0.0.22"
  ports = [
    "15432:5432"
  ]
  environment {
    POSTGRES_DB = "products"
    POSTGRES_USER = "postgres"
    POSTGRES_PASSWORD = "password"
  }
}
services "api" {
  environment {
    CONFIG_FILE = "/config/config.json"
  }
  depends_on = [
    "db"
  ]
  image = "hashicorpdemoapp/product-api:v0.0.22"
  ports = [
    "19090:9090"
  ]
  volumes = [
    "./conf.json:/config/config.json"
  ]
}
`)
	json, err := HCLToJSON(hcl)
	if err != nil {
		t.Fatalf("HCLToJSON failed: %v", err)
	}
	if len(json) == 0 {
		t.Error("Expected non-empty output")
	}
	output := string(json)
	if !containsString(output, `"version":"3.7"`) {
		t.Errorf("Expected '\"version\":\"3.7\"' in output, got:\n%s", output)
	}
}

// Test JSON to HCL conversion
func TestReadmeJSONToHCL(t *testing.T) {
	jsonData := []byte(`{"name":"test","value":42}`)
	hcl, err := JSONToHCL(jsonData)
	if err != nil {
		t.Fatalf("JSONToHCL failed: %v", err)
	}
	if len(hcl) == 0 {
		t.Error("Expected non-empty output")
	}
}

// Test YAML to HCL conversion
func TestReadmeYAMLToHCL(t *testing.T) {
	yamlData := []byte(`name: test
value: 42`)
	hcl, err := YAMLToHCL(yamlData)
	if err != nil {
		t.Fatalf("YAMLToHCL failed: %v", err)
	}
	if len(hcl) == 0 {
		t.Error("Expected non-empty output")
	}
}
