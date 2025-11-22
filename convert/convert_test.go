package convert

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/genelet/horizon/dethcl"
	"gopkg.in/yaml.v3"
)

func TestYaml2Json(t *testing.T) {
	testCases := []string{"x", "y", "z"}
	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			testYAMLConversions(t, tc)
		})
	}
}

func testYAMLConversions(t *testing.T, fn string) {
	raw, err := os.ReadFile(fn + ".yaml")
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}
	jsn, err := YAMLToJSON(raw)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}
	rawjson, err := os.ReadFile(fn + ".json")
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}
	if strings.TrimSpace(string(jsn)) != strings.TrimSpace(string(rawjson)) {
		t.Errorf("jsn: %s\n", jsn)
		t.Errorf("raw: %s\n", rawjson)
	}

	hcl, err := os.ReadFile(fn + ".hcl")
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}
	expected, err := YAMLToHCL(raw)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	hclmap := map[string]interface{}{}
	expectedmap := map[string]interface{}{}
	err = dethcl.Unmarshal(hcl, &hclmap)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}
	err = dethcl.Unmarshal(expected, &expectedmap)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}
	if !reflect.DeepEqual(hclmap, expectedmap) {
		t.Errorf("hcl: %#v\n", hclmap)
		t.Errorf("expected: %#v\n", expectedmap)
	}
}

func TestHcl2Json(t *testing.T) {
	testCases := []string{"x", "y", "z"}
	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			testHCLConversions(t, tc)
		})
	}
}

func testHCLConversions(t *testing.T, fn string) {
	raw, err := os.ReadFile(fn + ".hcl")
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}
	jsn, err := HCLToJSON(raw)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	rawjson, err := os.ReadFile(fn + ".json")
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}
	if strings.TrimSpace(string(jsn)) != strings.TrimSpace(string(rawjson)) {
		t.Errorf("jsn: %s\n", jsn)
		t.Errorf("raw: %s\n", rawjson)
	}

	rawyml, err := os.ReadFile(fn + ".yaml")
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}
	expected, err := HCLToYAML(raw)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	ymlmap := map[string]interface{}{}
	expectedmap := map[string]interface{}{}
	err = yaml.Unmarshal(rawyml, &ymlmap)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}
	err = yaml.Unmarshal(expected, &expectedmap)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}
	if !reflect.DeepEqual(ymlmap, expectedmap) {
		t.Errorf("yaml: %#v\n", ymlmap)
		t.Errorf("expected: %#v\n", expectedmap)
	}
}

func TestHcl2self(t *testing.T) {
	testCases := []string{"x", "y", "z"}
	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			testHCLRoundTrip(t, tc)
		})
	}
}

func testHCLRoundTrip(t *testing.T, fn string) {
	raw, err := os.ReadFile(fn + ".hcl")
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	hash := map[string]interface{}{}
	err = dethcl.Unmarshal(raw, &hash)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}
	bs, err := dethcl.Marshal(hash)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	hash1 := map[string]interface{}{}
	err = dethcl.Unmarshal(bs, &hash1)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}
	if !reflect.DeepEqual(hash, hash1) {
		t.Errorf("hash: %#v\n", hash)
		t.Errorf("hash1: %#v\n", hash1)
	}
}

// TestErrorCases tests error handling for invalid inputs
func TestErrorCases(t *testing.T) {
	testCases := []struct {
		name    string
		input   []byte
		convFn  func([]byte) ([]byte, error)
		wantErr bool
	}{
		{
			name:    "InvalidJSON_ToYAML",
			input:   []byte(`{invalid json`),
			convFn:  JSONToYAML,
			wantErr: true,
		},
		{
			name:    "InvalidJSON_ToHCL",
			input:   []byte(`{"unclosed": `),
			convFn:  JSONToHCL,
			wantErr: true,
		},
		{
			name:    "InvalidYAML_ToJSON",
			input:   []byte("invalid:\n  - yaml\n bad indent"),
			convFn:  YAMLToJSON,
			wantErr: true,
		},
		{
			name:    "InvalidYAML_ToHCL",
			input:   []byte(":: invalid yaml ::"),
			convFn:  YAMLToHCL,
			wantErr: true,
		},
		{
			name:    "InvalidHCL_ToJSON",
			input:   []byte(`block "unclosed" {`),
			convFn:  HCLToJSON,
			wantErr: true,
		},
		{
			name:    "InvalidHCL_ToYAML",
			input:   []byte(`invalid = = syntax`),
			convFn:  HCLToYAML,
			wantErr: true,
		},
		{
			name:    "EmptyInput_JSON",
			input:   []byte(``),
			convFn:  JSONToYAML,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.convFn(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
