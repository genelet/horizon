// Package dethcl implements marshaling and unmarshaling between HCL (HashiCorp Configuration Language) and Go structures.
//
// This package provides functionality similar to encoding/json but for HCL format.
// It extends the standard github.com/hashicorp/hcl/v2 library with support for:
//
//   - Interface types with dynamic resolution at runtime
//   - Maps with struct values (map[string]*Struct, map[[2]string]*Struct)
//   - Slices of interfaces ([]interface{})
//   - Nested structures with interface fields
//   - HCL labels for map keys
//   - Custom marshalers via Marshaler/Unmarshaler interfaces
//
// # Basic Usage
//
// For simple structures without interface fields:
//
//	type Config struct {
//	    Name    string   `hcl:"name"`
//	    Enabled bool     `hcl:"enabled,optional"`
//	    Tags    []string `hcl:"tags,optional"`
//	}
//
//	// Unmarshal
//	var cfg Config
//	err := dethcl.Unmarshal(hclBytes, &cfg)
//
//	// Marshal
//	bytes, err := dethcl.Marshal(&cfg)
//
// # Working with Interface Fields
//
// For structures containing interface fields, use UnmarshalSpec with type specifications:
//
//	type Shape interface {
//	    Area() float64
//	}
//
//	type Circle struct {
//	    Radius float64 `hcl:"radius"`
//	}
//
//	type Geo struct {
//	    Name  string `hcl:"name"`
//	    Shape Shape  `hcl:"shape,block"`
//	}
//
//	// Create specification telling how to decode the interface
//	spec, err := utils.NewStruct("Geo", map[string]interface{}{
//	    "Shape": "Circle",  // Shape field should be decoded as Circle type
//	})
//
//	// Reference map of available types
//	ref := map[string]interface{}{
//	    "Circle": &Circle{},
//	    "Geo":    &Geo{},
//	}
//
//	// Unmarshal with specification
//	var geo Geo
//	err = dethcl.UnmarshalSpec(hclBytes, &geo, spec, ref)
//
// # HCL Struct Tags
//
// The package uses struct tags to control marshaling/unmarshaling:
//
//   - `hcl:"name"` - Field name in HCL
//   - `hcl:"name,optional"` - Optional field (won't error if missing)
//   - `hcl:"name,block"` - Field is an HCL block (for structs, maps, slices)
//   - `hcl:"name,label"` - Field value becomes an HCL label (for map keys)
//   - `hcl:"-"` - Ignore this field
//
// # Map Encoding
//
// Maps are encoded as labeled blocks:
//
//	type Config struct {
//	    Services map[string]*Service `hcl:"service,block"`
//	}
//
// HCL output:
//
//	service "api" {
//	  port = 8080
//	}
//	service "db" {
//	  port = 5432
//	}
//
// # Custom Marshalers
//
// Implement Marshaler/Unmarshaler interfaces for custom encoding:
//
//	type Custom struct {
//	    data string
//	}
//
//	func (c *Custom) MarshalHCL() ([]byte, error) {
//	    return []byte(fmt.Sprintf("custom = %q", c.data)), nil
//	}
//
//	func (c *Custom) UnmarshalHCL(data []byte, labels ...string) error {
//	    // Custom parsing logic
//	    return nil
//	}
package dethcl
