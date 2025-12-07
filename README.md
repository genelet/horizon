# horizon

[![GoDoc](https://godoc.org/github.com/genelet/horizon?status.svg)](https://godoc.org/github.com/genelet/horizon)

`horizon` is a Go library that provides enhanced marshaling and unmarshaling capabilities between HCL (HashiCorp Configuration Language) and Go structures. It specifically addresses limitations in the standard HCL library regarding interface types and complex map structures.

It also includes utilities for converting between HCL, JSON, and YAML formats.

## Features

- **Enhanced HCL Support**:
  - **Interface Types**: Dynamic resolution of interface fields at runtime.
  - **Complex Maps**: Support for `map[string]*Struct` and `map[[2]string]*Struct`.
  - **Slices of Interfaces**: Handle `[]interface{}` seamlessly.
  - **HCL Labels**: Map keys can be used as HCL labels.
- **Format Conversion**: Convert data between HCL, JSON, and YAML.

## Installation

```bash
go get github.com/genelet/horizon
```

## Table of Contents

- Chapter 1: [Marshal Go Object into HCL](#chapter-1-marshal-go-object-into-hcl)
- Chapter 2: [Unmarshal HCL Data to Go Object](#chapter-2-unmarshal-hcl-data-to-go-object)
- Chapter 3: [Literals: true, false, and null](#chapter-3-literals-true-false-and-null)
- Chapter 4: [Functions and Function Calls in HCL](#chapter-4-functions-and-function-calls-in-hcl)
- Chapter 5: [Conversion among Data Formats HCL, JSON and YAML](#chapter-5-conversion-among-data-formats-hcl-json-and-yaml)

## Quick Start

The `dethcl` package provides `Marshal` and `Unmarshal` functions similar to `encoding/json`.

```go
package main

import (
    "fmt"
    "github.com/genelet/horizon/dethcl"
)

type Config struct {
    Name    string `hcl:"name"`
    Enabled bool   `hcl:"enabled,optional"`
}

func main() {
    // Unmarshal HCL
    hclData := []byte(`
        name = "example"
        enabled = true
    `)
    var cfg Config
    err := dethcl.Unmarshal(hclData, &cfg)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Config: %+v\n", cfg)

    // Marshal to HCL
    data, err := dethcl.Marshal(&cfg)
    if err != nil {
        panic(err)
    }
    fmt.Println(string(data))
}
```

### Command Line Tool (fmtconvert)

The project includes a command-line tool `fmtconvert` to convert between formats.

**Build:**

```bash
go build -o fmtconvert cmd/fmtconvert/main.go
```

**Usage:**

```bash
./fmtconvert -from json -to hcl input.json
```

Supported formats: `json`, `yaml`, `hcl`.

<br>

# Chapter 1. Marshal Go Object into HCL

## 1.1 Introduction

According to Hashicorp, HCL (Hashicorp Configuration Language) is a toolkit for creating structured configuration languages that are both human- and machine-friendly, for use with command-line tools. Whereas JSON and YAML are formats for serializing data structures, HCL is a syntax and API specifically designed for building structured configuration formats.

HCL is a key component of Hashicorp's cloud infrastructure automation tools, such as Terraform. Its robust support for configuration and expression syntax gives it the potential to serve as a server-side format. For instance, it could replace the backend programming language in low-code/no-code platforms. However, the current [HCL library](https://pkg.go.dev/github.com/hashicorp/hcl/v2) does not fully support some data types, such as _map_ and _interface_, which limits its usage.

## 1.2 Encoding Map

Here is an example to encode object with package _gohcl_:

```go
package main

import (
    "fmt"

    "github.com/hashicorp/hcl/v2/gohcl"
    "github.com/hashicorp/hcl/v2/hclwrite"
)

type square struct {
    SX int `json:"sx" hcl:"sx"`
    SY int `json:"sy" hcl:"sy"`
}

func (self *square) Area() float32 {
    return float32(self.SX * self.SY)
}

type geometry struct {
    Name   string              `json:"name" hcl:"name"`
    Shapes map[string]*square `json:"shapes" hcl:"shapes"`
}

func main() {
    app := &geometry{
        Name: "Medium Article",
        Shapes: map[string]*square{
            "k1": {SX: 2, SY: 3}, "k2": {SX: 5, SY: 6}},
    }

    f := hclwrite.NewEmptyFile()
    gohcl.EncodeIntoBody(app, f.Body())
    fmt.Printf("%s", f.Bytes())
}
```

It panics because of the map field `Shapes`:

```
panic: cannot encode map[string]*main.square as HCL expression: no cty.Type for main.square (no cty field tags)
```

But _horizon_ will encode it properly:

```go
package main

import (
    "fmt"

    "github.com/genelet/horizon/dethcl"
)

type square struct {
    SX int `json:"sx" hcl:"sx"`
    SY int `json:"sy" hcl:"sy"`
}

func (self *square) Area() float32 {
    return float32(self.SX * self.SY)
}

type geometry struct {
    Name   string              `json:"name" hcl:"name"`
    Shapes map[string]*square `json:"shapes" hcl:"shapes"`
}

func main() {
    app := &geometry{
        Name: "Medium Article",
        Shapes: map[string]*square{
            "k1": {SX: 2, SY: 3}, "k2": {SX: 5, SY: 6}},
    }

    bs, err := dethcl.Marshal(app)
    if err != nil {
        panic(err)
    }
    fmt.Printf("%s", bs)
}
```

Output:

```
name = "Medium Article"
shapes k1 {
  sx = 2
  sy = 3
}

shapes k2 {
  sx = 5
  sy = 6
}
```

> Note: map is encoded as block list with labels as keys.

## 1.3 Encode Interface Data

Go struct _picture_ has field `Drawings`, which is a list of _interface_. This sample shows how _horizon_ encodes data of one _square_ and one _circle_ in the list.

```go
package main

import (
    "fmt"
    "github.com/genelet/horizon/dethcl"
)

type inter interface {
    Area() float32
}

type square struct {
    SX int `json:"sx" hcl:"sx"`
    SY int `json:"sy" hcl:"sy"`
}

func (self *square) Area() float32 {
    return float32(self.SX * self.SY)
}

type circle struct {
    Radius float32 `json:"radius" hcl:"radius"`
}

func (self *circle) Area() float32 {
    return 3.14159 * self.Radius
}

type picture struct {
    Name     string  `json:"name" hcl:"name"`
    Drawings []inter `json:"drawings" hcl:"drawings"`
}

func main() {
    app := &picture{
        Name: "Medium Article",
        Drawings: []inter{
            &square{SX: 2, SY: 3}, &circle{Radius: 5.6}},
    }

    bs, err := dethcl.Marshal(app)
    if err != nil {
        panic(err)
    }
    fmt.Printf("%s", bs)
}
```

Output:

```
name = "Medium Article"
drawings {
  sx = 2
  sy = 3
}

drawings {
  radius = 5.6
}
```

## 1.4 Encoding with HCL Labels

`label` is encoded as map key. If it is missing, the block map will be encoded as list:

```go
package main

import (
    "fmt"
    "github.com/genelet/horizon/dethcl"
)

type inter interface {
    Area() float32
}

type square struct {
    SX int `json:"sx" hcl:"sx"`
    SY int `json:"sy" hcl:"sy"`
}

func (self *square) Area() float32 {
    return float32(self.SX * self.SY)
}

type moresquare struct {
    Morename1 string `json:"morename1" hcl:"morename1,label"`
    Morename2 string `json:"morename2" hcl:"morename2,label"`
    SX        int    `json:"sx" hcl:"sx"`
    SY        int    `json:"sy" hcl:"sy"`
}

func (self *moresquare) Area() float32 {
    return float32(self.SX * self.SY)
}

type picture struct {
    Name     string  `json:"name" hcl:"name"`
    Drawings []inter `json:"drawings" hcl:"drawings"`
}

func main() {
    app := &picture{
        Name: "Medium Article",
        Drawings: []inter{
            &square{SX: 2, SY: 3},
            &moresquare{Morename1: "abc2", Morename2: "def2", SX: 2, SY: 3},
        },
    }

    bs, err := dethcl.Marshal(app)
    if err != nil {
        panic(err)
    }
    fmt.Printf("%s", bs)
}
```

Output:

```
name = "Medium Article"
drawings {
  sx = 2
  sy = 3
}

drawings "abc2" "def2" {
  sx = 2
  sy = 3
}
```

The labels _abc2_ and _def2_ are properly placed in block `Drawings`.

## 1.5 Summary

The new HCL package, _horizon_, can marshal a wider range of Go objects, such as interfaces and maps, bringing HCL a step closer to becoming a universal data interchange format like JSON and YAML.

<br>

# Chapter 2. Unmarshal HCL Data to Go Object

## 2.1 Introduction

In this section, we will explore how to convert HCL data back into a Go object.

The _Unmarshal_ function in _horizon_ can:

- Support a wider range of data types, including map and labels
- Provide a powerful yet easy-to-use _Struct_ specification to decode data with a dynamic schema

Similar to JSON, HCL data cannot be decoded into an object if the latter contains an interface field. We need a specification for the actual data structure of the interface at runtime. HCL has the [_hcldec_](https://pkg.go.dev/github.com/hashicorp/hcl/v2) package to handle this issue.

However, _hcldec_ is not straightforward to use. For instance, describing the following data structure can be challenging:

```hcl
io_mode = "async"

service "http" "web_proxy" {
  listen_addr = "127.0.0.1:8080"

  process "main" {
    command = ["/usr/local/bin/awesome-app", "server", "gosh"]
    received = 1
  }

  process "mgmt" {
    command = ["/usr/local/bin/awesome-app", "mgmt"]
  }
}
```

_hcldec_ needs a long description:

```go
spec := hcldec.ObjectSpec{
    "io_mode": &hcldec.AttrSpec{
        Name: "io_mode",
        Type: cty.String,
    },
    "services": &hcldec.BlockMapSpec{
        TypeName:   "service",
        LabelNames: []string{"type", "name"},
        Nested:     hcldec.ObjectSpec{
            "listen_addr": &hcldec.AttrSpec{
                Name:     "listen_addr",
                Type:     cty.String,
                Required: true,
            },
            "processes": &hcldec.BlockMapSpec{
                TypeName:   "process",
                LabelNames: []string{"name"},
                Nested:     hcldec.ObjectSpec{
                    "command": &hcldec.AttrSpec{
                        Name:     "command",
                        Type:     cty.List(cty.String),
                        Required: true,
                    },
                },
            },
        },
    },
}
val, moreDiags := hcldec.Decode(f.Body, spec, nil)
diags = append(diags, moreDiags...)
```

> Note that _hcldec_ also parses variables, functions and expression evaluations, as we see in Terraform. Those features have only been implemented partially in _horizon_.

In _horizon_, the specification could be written simply using `schema.NewStruct` from the `github.com/genelet/schema` package:

```go
import "github.com/genelet/schema"

spec, err := schema.NewStruct("Terraform", map[string]any{
    "services": [][2]any{
        {"service", map[string]any{
            "processes": [2]any{
                "process", map[string]any{
                    "command": "commandName",
                }},
        }},
    },
})
```

which says that _service_ is the only item in list field `services`; within _service_, there is field `processes`, defined to be scalar of _process_, which contains interface field `command` and its runtime implementation is _commandName_. Fields of primitive data type or defined _go struct_ should be ignored in _spec_, because they will be decoded automatically.

## 2.2 Struct and Value

The `schema.NewStruct` function from `github.com/genelet/schema` is used to define interface structures. The function signature is:

```go
func NewStruct(class_name string, v ...map[string]any) (*Struct, error)
```

where v is a nested primitive map with:
- key being parsing tag of field name
- value being the following Struct conversions:

| Go type          | Conversion        |
|------------------|-------------------|
| string           | ending Struct     |
| [2]any           | SingleStruct      |
| []string         | ending ListStruct |
| [][2]any         | ListStruct        |
| *Struct          | SingleStruct      |
| []*Struct        | ListStruct        |

In the following example, the _geo_ type contains interface `Shape` which is implemented as either _circle_ or _square_:

```go
type geo struct {
    Name  string `json:"name" hcl:"name"`
    Shape inter  `json:"shape" hcl:"shape,block"`
}

type inter interface {
    Area() float32
}

type square struct {
    SX int `json:"sx" hcl:"sx"`
    SY int `json:"sy" hcl:"sy"`
}

func (self *square) Area() float32 {
    return float32(self.SX * self.SY)
}

type circle struct {
    Radius float32 `json:"radius" hcl:"radius"`
}

func (self *circle) Area() float32 {
    return 3.14159 * self.Radius
}
```

At run time, we know the data instance of geo is using type `Shape` = _circle_, so our _Struct_ is:

```go
import "github.com/genelet/schema"

spec, err := schema.NewStruct(
    "geo", map[string]any{"Shape": "circle"})
```

and for `Shape` of _square_:

```go
spec, err = schema.NewStruct(
    "geo", map[string]any{"Shape": "square"})
```

We have ignored field `Name` because it is a primitive type.

## 2.3 More Examples

Type _picture_ has field `Drawings` which is a list of `Shape` of size 2:

```go
type picture struct {
    Name     string  `json:"name" hcl:"name"`
    Drawings []inter `json:"drawings" hcl:"drawings,block"`
}
```

incoming data is slice of square, size 2:

```go
spec, err := schema.NewStruct(
    "Picture", map[string]any{
        "Drawings": []string{"square", "square"}})
```

Type _geometry_ has field `Shapes` as a map of `Shape` of size 2:

```go
type geometry struct {
    Name   string           `json:"name" hcl:"name"`
    Shapes map[string]inter `json:"shapes" hcl:"shapes,block"`
}

// incoming HCL data is map but MUST be expressed as slice of one label! e.g.
// name = "medium shapes"
//   shapes obj5 {
//     sx = 5
//     sy = 6
//   }
//   shapes obj7 {
//     sx = 7
//     sy = 8
//   }

spec, err := schema.NewStruct(
    "geometry", map[string]any{
        "Shapes": []string{"square", "square"}})
```

Type _toy_ has field `Geo` which contains `Shape`:

```go
type toy struct {
    Geo     geo     `json:"geo" hcl:"geo,block"`
    ToyName string  `json:"toy_name" hcl:"toy_name"`
    Price   float32 `json:"price" hcl:"price"`
}

spec, err = schema.NewStruct(
    "toy", map[string]any{
        "Geo": [2]any{
            "geo", map[string]any{"Shape": "square"}}})
```

Type _child_ has field `Brand` which is a map of the above _Nested of nested_ _toy_:

```go
type child struct {
    Brand map[string]*toy `json:"brand" hcl:"brand,block"`
    Age   int             `json:"age" hcl:"age"`
}

spec, err = schema.NewStruct(
    "child", map[string]any{
        "Brand": map[string][2]any{
            "abc1": {"toy", map[string]any{
                "Geo": [2]any{
                    "geo", map[string]any{"Shape": "circle"}}}},
            "def2": {"toy", map[string]any{
                "Geo": [2]any{
                    "geo", map[string]any{"Shape": "square"}}}},
        },
    },
)
```

## 2.4 Unmarshal HCL Data to Object

The decoding function _Unmarshal_ can be used in 4 cases.

1. Decode HCL _data_ to _object_ without dynamic schema:

```go
func Unmarshal(dat []byte, object any) error
```

2. Decode _data_ to _object_ without dynamic schema but with `label`. The labels will be assigned to the _label_ fields in _object_:

```go
func Unmarshal(dat []byte, object any, labels ...string) error
```

3. Decode _data_ to _object_ with dynamic schema specified by _spec_ and _ref_:

```go
func UnmarshalSpec(dat []byte, current any, spec *schema.Struct, ref map[string]any) error
//
// spec: describe how the interface fields are interpreted
// ref: a reference map to map class names in spec, to objects of empty value.
// e.g.
// ref := map[string]any{"circle": new(Circle), "geo": new(Geo)}
```

4. Decode _data_ to _object_ with dynamic schema specified by _spec_ and _ref_, and with `label`. The labels will be assigned to the _label_ fields in _object_:

```go
func UnmarshalSpec(dat []byte, current any, spec *schema.Struct, ref map[string]any, label_values ...string) error
```

In the following example, we decode data to _child_ of type _Nested of nested_, which contains multiple _interfaces_ and _maps_:

```go
package main

import (
    "fmt"

    "github.com/genelet/schema"
    "github.com/genelet/horizon/dethcl"
)

type inter interface {
    Area() float32
}

type square struct {
    SX int `json:"sx" hcl:"sx"`
    SY int `json:"sy" hcl:"sy"`
}

func (self *square) Area() float32 {
    return float32(self.SX * self.SY)
}

type circle struct {
    Radius float32 `json:"radius" hcl:"radius"`
}

func (self *circle) Area() float32 {
    return 3.14159 * self.Radius
}

type geo struct {
    Name  string `json:"name" hcl:"name"`
    Shape inter  `json:"shape" hcl:"shape,block"`
}

type toy struct {
    Geo     geo     `json:"geo" hcl:"geo,block"`
    ToyName string  `json:"toy_name" hcl:"toy_name"`
    Price   float32 `json:"price" hcl:"price"`
}

func (self *toy) ImportPrice(rate float32) float32 {
    return rate * 0.7 * self.Price
}

type child struct {
    Brand map[string]*toy `json:"brand" hcl:"brand,block"`
    Age   int             `json:"age" hcl:"age"`
}

func main() {
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
    spec, err := schema.NewStruct("child", map[string]any{
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
        panic(err)
    }
    ref := map[string]any{"toy": &toy{}, "geo": &geo{}, "circle": &circle{}, "square": &square{}}

    c := new(child)
    err = dethcl.UnmarshalSpec([]byte(data1), c, spec, ref)
    if err != nil {
        panic(err)
    }
    fmt.Printf("%v\n", c.Age)
    fmt.Printf("%#v\n", c.Brand["abc1"])
    fmt.Printf("%#v\n", c.Brand["abc1"].Geo.Shape)
    fmt.Printf("%#v\n", c.Brand["def2"])
    fmt.Printf("%#v\n", c.Brand["def2"].Geo.Shape)
}
```

Output:

```
5
&main.toy{Geo:main.geo{Name:"medium shape", Shape:(*main.circle)(0xc000018650)}, ToyName:"roblox", Price:99.9}
&main.circle{Radius:1.234}
&main.toy{Geo:main.geo{Name:"square shape", Shape:(*main.square)(0xc000018890)}, ToyName:"minecraft", Price:9.9}
&main.square{SX:5, SY:6}
```

The output is populated properly into specified objects.

## 2.5 Enhanced UnmarshalSpec with Auto-Discovery

The `UnmarshalSpec` function has been enhanced to automatically discover struct types, reducing the need for manual `ref` map construction.

**Auto-Discovery Behavior:**

`UnmarshalSpec` now internally auto-discovers struct types from the target object, so concrete struct types are automatically found. You only need to provide:
1. Types that cannot be auto-discovered (interface implementations)
2. Explicit overrides for type resolution

**Passing Implementations:**

The `ref` map can include `[]any` values to specify interface implementations:

```go
ref := map[string]any{
    // Interface implementations ([]any values)
    "Shape": []any{new(Circle), new(Square)},

    // Explicit type overrides (optional)
    "CustomType": new(MyCustomType),
}

err := dethcl.UnmarshalSpec(hclData, &config, spec, ref)
```

**Simplified Usage Example:**

Before (manual ref construction):

```go
ref := map[string]any{
    "Config":   new(Config),
    "Team":     new(Team),
    "Auth":     new(Auth),
    "DBIssuer": new(DBIssuer),
    // ... many more types
}
err := dethcl.UnmarshalSpec(data, &config, spec, ref)
```

After (with auto-discovery):

```go
// Only specify interface implementations
ref := map[string]any{
    "Squad":         []any{new(Team)},
    "Authenticator": []any{new(Auth)},
    "Issuer":        []any{new(DBIssuer), new(PlainIssuer)},
}
err := dethcl.UnmarshalSpec(data, &config, spec, ref)
```

**How it works internally:**
1. Starts from the target object and traverses all fields recursively
2. For struct fields: adds the struct type to the internal ref map
3. For interface fields: looks up the implementations from `[]any` values in the passed ref
4. For map/slice fields: processes the element type recursively
5. Adds both short names and package-qualified names for each type

**Important Notes:**
- Go reflection cannot discover which types implement an interface, so you must provide interface implementations as `[]any` values in the ref map
- Package-qualified names (e.g., `"cell.Config"`) are automatically added alongside short names
- Explicitly passed ref values take precedence over auto-discovered types

<br>

# Chapter 3. Literals: true, false, and null

## 3.1 Introduction

HCL supports three special literal values: `true`, `false`, and `null`. These literals work similarly to their counterparts in JSON and other programming languages, but with some specific behaviors in the `horizon` library.

## 3.2 Boolean Literals: true and false

Boolean values in HCL are represented by the lowercase keywords `true` and `false`. They map directly to Go's `bool` type.

**Example HCL:**

```hcl
enabled = true
disabled = false
renewable = true
```

**Go struct:**

```go
type Config struct {
    Enabled   bool `hcl:"enabled"`
    Disabled  bool `hcl:"disabled"`
    Renewable bool `hcl:"renewable"`
}
```

When unmarshaling, `true` becomes Go's `true` and `false` becomes Go's `false`. When marshaling, Go boolean values are converted back to their HCL literal equivalents.

## 3.3 The null Literal

The `null` literal represents the absence of a value. In `horizon`, `null` has special handling:

**Marshaling behavior:**
- Go `nil` pointers, interfaces, maps, slices, and functions are marshaled as `null`
- Example: A `nil` map field becomes `data = null` in HCL

**Unmarshaling behavior:**
- When `null` is encountered, the corresponding Go field retains its zero value
- Fields with `null` values are tracked and skipped during struct population
- This allows distinguishing between "not set" and "explicitly null"

**Example HCL with null:**

```hcl
body_data {
    renewable = false
    lease_duration = 0
    data = null
    wrap_info = null
    auth {
        client_token = "hvs.xxx"
        mfa_requirement = null
        num_uses = 0
    }
}
```

**Go struct:**

```go
type Response struct {
    BodyData map[string]any `hcl:"body_data,block"`
}
```

When this HCL is unmarshaled, fields like `data`, `wrap_info`, and `mfa_requirement` will be `nil` in the resulting Go map.

## 3.4 Using null() as a Function

In addition to the literal `null`, `horizon` supports `null()` as a function call that returns a null value. This can be useful in expressions:

```hcl
optional_value = null()
```

Both `null` and `null()` produce the same result.

## 3.5 Practical Example

Here's a complete example showing how literals are handled:

```go
package main

import (
    "fmt"
    "github.com/genelet/horizon/dethcl"
)

type AuthInfo struct {
    ClientToken    string   `hcl:"client_token"`
    Policies       []string `hcl:"policies"`
    Renewable      bool     `hcl:"renewable"`
    MfaRequirement any      `hcl:"mfa_requirement"`
}

type Response struct {
    RequestID     string            `hcl:"request_id"`
    LeaseID       string            `hcl:"lease_id"`
    Renewable     bool              `hcl:"renewable"`
    LeaseDuration int               `hcl:"lease_duration"`
    Data          map[string]any    `hcl:"data,block"`
    Auth          *AuthInfo         `hcl:"auth,block"`
}

func main() {
    hclData := `
request_id = "2e7a9b1d-a8d6-4ce4-6380-47c05cf1d16e"
lease_id = ""
renewable = false
lease_duration = 0
data = null

auth {
    client_token = "hvs.secret"
    policies = ["default", "admin"]
    renewable = true
    mfa_requirement = null
}
`
    var resp Response
    err := dethcl.Unmarshal([]byte(hclData), &resp)
    if err != nil {
        panic(err)
    }

    fmt.Printf("RequestID: %s\n", resp.RequestID)
    fmt.Printf("Renewable: %v\n", resp.Renewable)        // false
    fmt.Printf("LeaseDuration: %d\n", resp.LeaseDuration) // 0
    fmt.Printf("Data is nil: %v\n", resp.Data == nil)     // true (was null)
    fmt.Printf("Auth.Renewable: %v\n", resp.Auth.Renewable) // true
    fmt.Printf("Auth.MfaRequirement is nil: %v\n", resp.Auth.MfaRequirement == nil) // true
}
```

Output:

```
RequestID: 2e7a9b1d-a8d6-4ce4-6380-47c05cf1d16e
Renewable: false
LeaseDuration: 0
Data is nil: true
Auth.Renewable: true
Auth.MfaRequirement is nil: true
```

## 3.6 Summary

- `true` and `false` are boolean literals that map to Go's `bool` type
- `null` represents absence of value and maps to Go's `nil`
- During marshaling, `nil` values become `null` in HCL output
- During unmarshaling, `null` values result in `nil` or zero values in Go
- Both the literal `null` and function `null()` are supported

<br>

# Chapter 4. Functions and Function Calls in HCL

## 4.1 Introduction

One of HCL's powerful features is its support for expressions, variables, and function calls. Unlike JSON and YAML which are purely declarative, HCL allows dynamic values computed at parse time. The `horizon` library extends this capability by allowing you to define and use custom functions during HCL parsing.

This chapter explains how to:
- Define custom functions for use in HCL configurations
- Reference variables across the configuration
- Use string interpolation with computed values

## 4.2 Defining Custom Functions

Custom functions can be passed to `UnmarshalSpec` through the `ref` map using the key `"functions"`. There are two ways to define functions:

### Method 1: Using Native Go Functions

The simplest approach is to pass regular Go functions. The library will automatically handle argument conversion and return value processing:

```go
ref := map[string]any{
    "functions": map[string]any{
        "random": func(n int) string {
            var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
            b := make([]rune, n)
            for i := range b {
                b[i] = letterRunes[rand.Intn(len(letterRunes))]
            }
            return string(b)
        },
    },
}
```

Functions can also return errors as a second return value:

```go
ref := map[string]any{
    "functions": map[string]any{
        "datetimeparse": func(layout, value string) (int64, error) {
            t, err := time.Parse(layout, value)
            if err != nil {
                return 0, err
            }
            return t.Unix(), nil
        },
    },
}
```

### Method 2: Using cty Functions

For more control over type handling, you can use the `cty/function` package directly:

```go
import (
    "github.com/zclconf/go-cty/cty"
    "github.com/zclconf/go-cty/cty/function"
)

ref := map[string]any{
    "functions": map[string]function.Function{
        "random": function.New(&function.Spec{
            Params: []function.Parameter{
                {Type: cty.Number},
            },
            Type: func(args []cty.Value) (cty.Type, error) {
                return cty.String, nil
            },
            Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
                var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
                n, _ := args[0].AsBigFloat().Int64()
                b := make([]rune, n)
                for i := range b {
                    b[i] = letterRunes[rand.Intn(len(letterRunes))]
                }
                return cty.StringVal(string(b)), nil
            },
        }),
    },
}
```

## 4.3 Variable References and Expressions

HCL in `horizon` supports:

- **Variable references**: Use `var.FieldName` to reference other fields
- **Arithmetic expressions**: Use operators like `+`, `-`, `*`, `/`
- **String interpolation**: Use `${expression}` within strings
- **For expressions**: Use `for k, v in collection: k => v if condition`

## 4.4 Complete Example

Here is a complete example demonstrating functions, variables, and expressions (from `dethcl/dyna_test.go`):

```go
package main

import (
    "fmt"
    "math/rand"

    "github.com/genelet/horizon/dethcl"
)

type Slack struct {
    Channel string `hcl:"channel"`
    Message string `hcl:"message"`
}

type Python struct {
    PythonName    string `hcl:"python_name,label"`
    PythonVersion int    `hcl:"python_version,optional"`
    Path          string `hcl:"root_dir,optional"`
}

type Job struct {
    JobName       string  `hcl:"job_name,label"`
    Description   string  `hcl:"description,label"`
    ProgramPython *Python `hcl:"python,block"`
    ProgramSlack  *Slack  `hcl:"slack,block"`
}

type Pipeline struct {
    Version     int               `hcl:"version,optional"`
    Say         map[string]string `hcl:"say,optional"`
    TestFolder  string            `hcl:"TestFolder"`
    ExecutionID string            `hcl:"ExecutionID"`
    Jobs        []*Job            `hcl:"job,block"`
}

func main() {
    hclData := `
TestFolder = "__test__"
ExecutionID = random(6)
version = 2
say = {
    for k, v in {hello: "world"}: k => v if k == "hello"
}

job check "this is a temporal job" {
    python "run.py" {}
}

job e2e "running integration tests" {
    python "app-e2e.py" {
        root_dir = var.TestFolder
        python_version = version + 6
    }

    slack {
        channel  = "slack-my-channel"
        message = "Job execution ${ExecutionID} completed successfully"
    }
}
`

    p := new(Pipeline)
    ref := map[string]any{
        "functions": map[string]any{
            "random": func(n int) string {
                var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
                b := make([]rune, n)
                for i := range b {
                    b[i] = letterRunes[rand.Intn(len(letterRunes))]
                }
                return string(b)
            },
        },
    }

    err := dethcl.UnmarshalSpec([]byte(hclData), p, nil, ref)
    if err != nil {
        panic(err)
    }

    fmt.Printf("TestFolder: %s\n", p.TestFolder)
    fmt.Printf("ExecutionID: %s (random 6-char string)\n", p.ExecutionID)
    fmt.Printf("Version: %d\n", p.Version)
    fmt.Printf("Say: %v\n", p.Say)

    for _, job := range p.Jobs {
        fmt.Printf("\nJob: %s - %s\n", job.JobName, job.Description)
        if job.ProgramPython != nil {
            fmt.Printf("  Python: %s, version=%d, path=%s\n",
                job.ProgramPython.PythonName,
                job.ProgramPython.PythonVersion,
                job.ProgramPython.Path)
        }
        if job.ProgramSlack != nil {
            fmt.Printf("  Slack: channel=%s, message=%s\n",
                job.ProgramSlack.Channel,
                job.ProgramSlack.Message)
        }
    }
}
```

This example demonstrates:

1. **Custom function call**: `ExecutionID = random(6)` calls the custom `random` function
2. **For expression**: `say = { for k, v in {hello: "world"}: k => v if k == "hello" }` filters a map
3. **Variable reference**: `root_dir = var.TestFolder` references another field
4. **Arithmetic expression**: `python_version = version + 6` computes `2 + 6 = 8`
5. **String interpolation**: `"Job execution ${ExecutionID} completed successfully"` embeds the computed value
6. **Multiple labels**: `job e2e "running integration tests"` has two labels (`job_name` and `description`)

Output:

```
TestFolder: __test__
ExecutionID: xKmPqR (random 6-char string)
Version: 2
Say: map[hello:world]

Job: check - this is a temporal job
  Python: run.py, version=0, path=

Job: e2e - running integration tests
  Python: app-e2e.py, version=8, path=__test__
  Slack: channel=slack-my-channel, message=Job execution xKmPqR completed successfully
```

## 4.5 Built-in Functions

The `horizon` library includes a set of built-in functions similar to Terraform, located in `internal/lang/funcs`. These include:

- **String functions**: `upper`, `lower`, `trim`, `replace`, `split`, `join`, etc.
- **Collection functions**: `length`, `element`, `keys`, `values`, `merge`, `flatten`, etc.
- **Numeric functions**: `abs`, `ceil`, `floor`, `min`, `max`, etc.
- **Encoding functions**: `base64encode`, `base64decode`, `jsonencode`, `jsondecode`, etc.
- **Crypto functions**: `md5`, `sha1`, `sha256`, `sha512`, `uuid`, etc.
- **Date/time functions**: `timestamp`, `formatdate`, `timeadd`, etc.
- **Filesystem functions**: `file`, `fileexists`, `basename`, `dirname`, etc.
- **CIDR functions**: `cidrhost`, `cidrnetmask`, `cidrsubnet`, etc.

## 4.6 Summary

HCL's expression and function support makes it much more powerful than static configuration formats like JSON and YAML. With `horizon`, you can:

- Define custom functions to extend HCL's capabilities
- Use variables to avoid repetition and ensure consistency
- Compute values dynamically at parse time
- Create pipeline-style configurations with complex logic

This makes HCL an excellent choice for infrastructure-as-code, CI/CD pipelines, and other configuration scenarios where dynamic values are needed.

<br>

# Chapter 5. Conversion among Data Formats HCL, JSON and YAML

## 5.1 Introduction

Hashicorp Configuration Language ([HCL](https://github.com/hashicorp/hcl)) is a user-friendly data format for structured configuration. It combines parameters and declarative logic in a way that is easily understood by both humans and machines. HCL is integral to Hashicorp's cloud infrastructure automation tools, such as `Terraform` and `Nomad`. With its robust support for expression syntax, HCL has the potential to serve as a general data format with programming capabilities, making it suitable for use in no-code platforms.

However, in many scenarios, we still need to use popular data formats like JSON and YAML alongside HCL. For instance, Hashicorp products use JSON for data communication via REST APIs, while Docker or Kubernetes management in `Terraform` requires YAML.

## 5.2 Question

An intriguing question arises: Is it possible to convert HCL to JSON or YAML, and vice versa? Could we use HCL as the universal configuration language in projects and generate YAML or JSON with CLI or within `Terraform` on the fly?

Unfortunately, the answer is generally no. The expressive power of HCL surpasses that of JSON and YAML. In particular, HCL uses array key (i.e. labels) to express maps, while JSON and YAML use single maps. Most importantly, HCL allows variables and logic expressions, while JSON and YAML are purely data declarative. Therefore, some features in HCL can never be accurately represented in JSON.

However, in cases where we don't care about map orders, and there are no variables or logical expressions, but only generic maps, lists, and scalars, then the answer is yes. This type of HCL can be accurately converted to JSON, and vice versa.

> There is a practical advantage of HCL over YAML: HCL is very readable and less prone to errors, while YAML is sensitive to markers like white-space. One can write a configuration in HCL and let a program handle conversion.

## 5.3 The Package

`horizon` is a Go package to marshal and unmarshal dynamic JSON and HCL contents with interface types. It has a `convert` library for conversions among different data formats.

Technically, a JSON or YAML string can be unmarshalled into an anonymous map of _map[string]any_. For seamless conversion, `horizon` has internally implemented methods to unmarshal any HCL string into an anonymous map, and marshal an anonymous map into a properly formatted HCL string.

The following functions in `horizon/convert` can be used for conversion:

- hcl to json: `HCLToJSON(raw []byte) ([]byte, error)`
- hcl to yaml: `HCLToYAML(raw []byte) ([]byte, error)`
- json to hcl: `JSONToHCL(raw []byte) ([]byte, error)`
- json to yaml: `JSONToYAML(raw []byte) ([]byte, error)`
- yaml to hcl: `YAMLToHCL(raw []byte) ([]byte, error)`
- yaml to json: `YAMLToJSON(raw []byte) ([]byte, error)`

If you start with HCL, make sure it contains only primitive data types of maps, lists and scalars.

> In HCL, square brackets are lists and curly brackets are maps. Use **equal sign `=`** and **comma** to separate values for **list** assignment. But no equal sign nor comma for map.

Here is the example to convert HCL to YAML:

```go
package main

import (
    "fmt"
    "github.com/genelet/horizon/convert"
)

func main() {
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
    yml, err := convert.HCLToYAML(bs)
    if err != nil {
        panic(err)
    }
    fmt.Printf("%s\n", yml)
}
```

> Note that HCL is enclosed internally in curly bracket. But the top-level curly bracket should be removed, so it can be accepted by [the HCL parser](https://pkg.go.dev/github.com/hashicorp/hcl/v2/hclsyntax).

Output:

```yaml
name: marcus
num: 2
parties:
    - one
    - two
    - - three
      - four
    - five: "51"
      six: 61
radius: 1
roads:
    x: a
    xy:
        - ab
        - true
    "y": b
    z:
        za: aa
        zb: 3.14
```

## 5.4 The CLI

In directory `cmd`, there is a CLI program `fmtconvert`. Its usage is:

```bash
$ go run cmd/fmtconvert/main.go

fmtconvert [options] <filename>
  -from string
     from format (default "hcl")
  -to string
     to format (default "yaml")
```

This is a HCL:

```hcl
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
```

Convert it to JSON:

```bash
$ go run cmd/fmtconvert/main.go -to json the_above.hcl

{"services":{"api":{"depends_on":["db"],"environment":{"CONFIG_FILE":"/config/config.json"},"image":"hashicorpdemoapp/product-api:v0.0.22","ports":["19090:9090"],"volumes":["./conf.json:/config/config.json"]},"db":{"environment":{"POSTGRES_DB":"products","POSTGRES_PASSWORD":"password","POSTGRES_USER":"postgres"},"image":"hashicorpdemoapp/product-api-db:v0.0.22","ports":["15432:5432"]}},"version":"3.7"}
```

Convert it to YAML:

```bash
$ go run cmd/fmtconvert/main.go the_above.hcl

services:
    api:
        depends_on:
            - db
        environment:
            CONFIG_FILE: /config/config.json
        image: hashicorpdemoapp/product-api:v0.0.22
        ports:
            - 19090:9090
        volumes:
            - ./conf.json:/config/config.json
    db:
        environment:
            POSTGRES_DB: products
            POSTGRES_PASSWORD: password
            POSTGRES_USER: postgres
        image: hashicorpdemoapp/product-api-db:v0.0.22
        ports:
            - 15432:5432
version: "3.7"
```

We see that HCL's syntax is cleaner, more readable, and less error-prone compared to JSON and YAML.

## 5.5 Summary

HCL is a novel data format that offers advantages over JSON and YAML. In this article, we have demonstrated how to convert data among these three formats.

## License

See [LICENSE](LICENSE) file.
