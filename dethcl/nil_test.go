package dethcl

import "testing"

type testConfig struct {
	Name string `hcl:"name"`
}

func TestUnmarshalNilPointer(t *testing.T) {
	var cfg *testConfig
	if err := Unmarshal([]byte(`name = "demo"`), cfg); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestMarshalNilPointer(t *testing.T) {
	var cfg *testConfig
	got, err := Marshal(cfg)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil output, got %q", string(got))
	}
}

func TestUnmarshalNilMap(t *testing.T) {
	var m map[string]any
	if err := Unmarshal([]byte(`foo = "bar"`), &m); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if m["foo"] != "bar" {
		t.Fatalf("expected foo=bar, got %#v", m)
	}
}
