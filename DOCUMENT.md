# horizon

_Determined_ marshals and unmarshals JSON and HCL data to _go struct_ containing interfaces horizon at run-time.

[![GoDoc](https://godoc.org/github.com/genelet/horizon?status.svg)](https://godoc.org/github.com/genelet/horizon)

- Chapter 1: [Marshal GO Object into HCL](#chapter-1-marshal-go-object-into-hcl) (for encoding HCL object)
- Chapter 2: [Unmarshal HCL Data to GO Object](#chapter-2-unmarshal-hcl-data-to-go-object) (for dynamic HCL decoding)
  - 2.5: [Enhanced UnmarshalSpec with Auto-Discovery](#25-enhanced-unmarshalspec-with-auto-discovery)
- Chapter 3: [Conversion among Data Formats HCL, JSON and YAML](#chapter-3-conversion-among-data-formats-hcl-json-and-yaml)

To download,

```bash
go get github.com/genelet/horizon
```

![](https://miro.medium.com/v2/resize:fit:933/1*xn5HOalL1t6MPN654vynGQ.png)

<br>

<br>

# Chapter 1. Marshal GO Object into HCL
<br>

## 1.1 Introduction

According to Hashicorp, HCL (Hashicorp Configuration Language) is a toolkit for creating structured configuration languages that are both human- and machine-friendly, for use with command-line tools. Whereas JSON and YAML are formats for serializing data structures, HCL is a syntax and API specifically designed for building structured configuration formats.

HCL is a key component of Hashicorp's cloud infrastructure automation tools, such as Terraform. Its robust support for configuration and expression syntax gives it the potential to serve as a server-side format. For instance, it could replace the backend programming language in low-code/no-code platforms. However, the current  [HCL library](https://pkg.go.dev/github.com/hashicorp/hcl/v2)  does not fully support some data types, such as  _map_  and  _interface_, which limits its usage.

## 1.2 Encoding Map

Here is an example to encode object with package  _gohcl._
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
        Name   string       `json:"name" hcl:"name"`  
        Shapes map[string]*square `json:"shapes" hcl:"shapes"`  
    }  
      
    func main() {  
        app := &geometry{  
            Name: "Medium Article",  
            Shapes: map[string]*square{  
                "k1": &square{SX: 2, SY: 3}, "k2": &square{SX: 5, SY: 6}},  
        }  
      
        f := hclwrite.NewEmptyFile()  
        gohcl.EncodeIntoBody(app, f.Body())  
        fmt.Printf("%s", f.Bytes())  
    }
```
It panics because of the map field  `Shapes`.

    panic: cannot encode map[string]*main.square as HCL expression: no cty.Type for main.square (no cty field tags)

But  _horizon_ will encode it properly:
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
    Name   string       `json:"name" hcl:"name"`
    Shapes map[string]*square `json:"shapes" hcl:"shapes"`
}

func main() {
    app := &geometry{
        Name: "Medium Article",
        Shapes: map[string]*square{
            "k1": &square{SX: 2, SY: 3}, "k2": &square{SX: 5, SY: 6}},
    }

    bs, err := dethcl.Marshal(app)
    if err != nil {
        panic(err)
    }
    fmt.Printf("%s", bs)
}
```
Run the code:
```bash
    $ go run sample1_2.go  
      
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
Note:

> map is encoded as block list with labels as keys.

## 2.3 Encode Interface Data

Go struct  _picture_  has field  `Drawings`, which is a list of  _interface_. This sample shows how  _horizon_  encodes data of one  _square_  and one  _circle_  in the list.
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
        Name   string    `json:"name" hcl:"name"`  
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
Run the code:
```bash
    $ go run sample1_3.go   
      
    name = "Medium Article"  
    drawings {  
     sx = 2  
     sy = 3  
    }  
      
    drawings {  
     radius = 6  
    }
```
## 2.4  Encoding with HCL Labels

`label`  is encoded as map key. If it is missing, the block map will be encoded as list:
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
    Morename1 string `json:"morename1", hcl:"morename1,label"`
    Morename2 string `json:"morename2", hcl:"morename2,label"`
    SX int `json:"sx" hcl:"sx"`
    SY int `json:"sy" hcl:"sy"`
}

func (self *moresquare) Area() float32 {
    return float32(self.SX * self.SY)
}

type picture struct {
    Name   string    `json:"name" hcl:"name"`
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
Run the code:
```bash
    $ go run sample1_5.go   
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
The labels  _abc2_  and  _def2_  are properly placed in block  `Drawings`.

## 2.5  Summary

The new HCL package,  _horizon_, can marshal a wider range of Go objects, such as interfaces and maps, bringing HCL a step closer to becoming a universal data interchange format like JSON and YAML.

<br>

<br>

# Chapter 2. Unmarshal HCL Data to GO Object
<br>

## 2.1 Introduction

In this section, we will explore how to convert HCL data back into a Go object.

The  _Unmarshal_  function in  _horizon_  can

-   support a wider range of data types, including map and labels
-   provide a powerful yet easy-to-use  _Struct_  specification to decode data with a dynamic schema

Similar to JSON, HCL data cannot be decoded into an object if the latter contains an interface field. We need a specification for the actual data structure of the interface at runtime. HCL has the  [_hcldec_](https://pkg.go.dev/github.com/hashicorp/hcl/v2)  package to handle this issue.

However,  _hcldec_  is not straightforward to use. For instance, describing the following data structure can be challenging:
```bash
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
_hcldec_  needs a long description:
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
> Note that  _hcldec_  also parses variables, functions and expression evaluations, as we see in Terraform. Those features have only been implemented partially in  _horizon_.

In  _horizon_, the specification could be written simply as:
```go
    spec, err := NewStruct("Terraform", map[string]interface{}{  
      "services": [][2]interface{}{  
        {"service", map[string]interface{}{  
          "processes": [2]interface{}{  
            "process", map[string]interface{}{  
              "command": "commandName",  
            }},  
          },  
        }},  
      },  
    } 
```
which says that  _service_  is the only item in list field  `services`; within  _service_, there is field  `processes`, defined to be scalar of  _process_, which contains interface field  `command`  and its runtime implementation is  _commandName_. Fields of primitive data type or defined  _go struct_  should be ignored in  _spec_, because they will be decoded automatically.

## 2.2 Struct and Value

Beneath the surface, we have followed Go’s  _reflect_  package to define data  _Struct_  and  _Value_  in proto message,
```bash
    syntax = "proto3";  
      
    package dethcl;  
      
    option go_package = "./dethcl";  
      
    message Struct {  
      string className = 1;  
      map<string, Value> fields = 2;  
    }  
      
    message Value {  
      // The kind of value.  
      oneof kind {  
        Struct single_struct   = 1;  
        ListStruct list_struct = 2;  
      }  
    }  
      
    message ListStruct {  
      repeated Struct list_fields = 1;  
    }
 ```   

which is auto generated into the Go code:

  ```go  
    type Struct struct {  
        ClassName string            `protobuf:"bytes,1,opt,name=className,proto3" json:"className,omitempty"`  
        Fields    map[string]*Value `protobuf:"bytes,2,rep,name=fields,proto3" json:"fields,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`  
    }  
      
    type Value struct {  
        // The kind of value.  
        //  
        // Types that are assignable to Kind:  
        //  
        //  *Value_SingleStruct  
        //  *Value_ListStruct  
        Kind isValue_Kind `protobuf_oneof:"kind"`  
    }  
      
    type ListStruct struct {  
        ListFields []*Struct `protobuf:"bytes,1,rep,name=list_fields,json=listFields,proto3" json:"list_fields,omitempty"`  
    }  
      
    ...
```
To build a new  _Struct_, use function  _NewStruct_:
```go
    func NewStruct(class_name string, v …map[string]interface{}) (*Struct, error)  
    //  
    // where v is a nested primative map with  
    // - key being parsing tag of field name  
    // - value being the following Struct conversions:  
    //  
    //  ╔══════════════════╤═══════════════════╗  
    //  ║ Go type          │ Conversion        ║  
    //  ╠══════════════════╪═══════════════════╣  
    //  ║ string           │ ending Struct     ║  
    //  ║ [2]interface{}   │ SingleStruct      ║  
    //  ║                  │                   ║  
    //  ║ []string         │ ending ListStruct ║  
    //  ║ [][2]interface{} │ ListStruct        ║  
    //  ║                  │                   ║  
    //  ║ *Struct          │ SingleStruct      ║  
    //  ║ []*Struct        │ ListStruct        ║  
    //  ╚══════════════════╧═══════════════════╝
```
In the following example, the  _geo_  type contains interface  `Shape`  which is implemented as either  _circle_  or  _square_:
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

At run time, we know the data instance of geo is using type  `Shape`  =  _cirle_, so our  _Struct_  is:

```go
    spec, err := dethcl.NewStruct(  
      "geo", map[string]interface{}{"Shape": "circle"})
``` 

and for  `Shape`  of  _square_:

 ```go   
    spec, err = NewStruct(  
      "geo", map[string]interface{}{"Shape": "square"})
```
We have ignored field  `Name`  because it is a primitive type.

## 2.3 More Examples

Type _picture_  has field  `Drawings`  which is a list of  `Shape`  of size 2:
```go
    type picture struct {  
        Name     string   `json:"name" hcl:"name"`  
        Drawings []inter  `json:"drawings" hcl:"drawings,block"`  
    }  
```
  
incoming data is slice of square, size 2  
```go
    spec, err := NewStruct(  
      "Picture", map[string]interface{}{  
        "Drawings": []string{"square", "square"}})
```

Type  _geometry_  has field  `Shapes`  as a map of  `Shape`  of size 2:

```go
type geometry struct {
    Name   string           `json:"name" hcl:"name"`
    Shapes map[string]inter `json:"shapes" hcl:"shapes,block"`
}

# incoming HCL data is map but MUST be expressed as slice of one label! e.g.
# name = "medium shapes"
#   shapes obj5 {
#     sx = 5
#     sy = 6
#   }
#   shapes obj7 {
#     sx = 7
#     sy = 8
#   }

spec, err := NewStruct(
  "geometry", map[string]interface{}{
    "Shapes": []string{"square", "square"}})
```
  
Type  _toy_  has field`Geo`  which contains  `Shape`:

```go
    type toy struct {  
        Geo     geo     `json:"geo" hcl:"geo,block"`  
        ToyName string  `json:"toy_name" hcl:"toy_name"`  
        Price   float32 `json:"price" hcl:"price"`  
    }  
      
    spec, err = NewStruct(  
      "toy", map[string]interface{}{  
        "Geo": [2]interface{}{  
          "geo", map[string]interface{}{"Shape": "square"}}})
```

Type  _child_  has field  `Brand`  which is a map of the above  _Nested of nested_  _toy_:

```go
    type child struct {  
        Brand map[string]*toy `json:"brand" hcl:"brand,block"`  
        Age   int  `json:"age" hcl:"age"`  
    }  
      
    spec, err = NewStruct(  
      "child", map[string]interface{}{  
        "Brand": map[string][2]interface{}{  
          "abc1": {"toy", map[string]interface{}{  
            "Geo": [2]interface{}{  
              "geo", map[string]interface{}{"Shape": "circle"}}}},  
          "ref2": {"toy", map[string]interface{}{  
            "Geo": [2]interface{}{  
              "geo", map[string]interface{}{"Shape": "square"}}}},  
        },  
      },  
    )
```

## 2.4 Unmarshal HCL Data to Object

The decoding function  _Unmarshal_  can be used in 4 cases.

1.  Decode HCL  _data_  to  _object_  without dynamic schema.
```go
    func Unmarshal(dat []byte, object interface{}) error
```
2. Decode  _data_  to  _object_  without dynamic schema but with  `label`. The labels will be assigned to the  _label_  fields in  _object_.
```go
    func Unmarshal(dat []byte, object interface{}, labels ...string) error
```

3. Decode  _data_  to  _object_  with dynamic schema specified by  _spec_  and  _ref_.
```go
func UnmarshalSpec(dat []byte, current interface{}, spec *Struct, ref map[string]interface{}) error 
//
// spec: describe how the interface fields are interprested
// ref: a reference map to map class names in spec, to objects of empty value.
// e.g.
// ref := map[string]interface{}{"cirle": new(Circle), "geo": new(Geo)}
```

4. Decode  _data_  to  _object_  with dynamic schema specified by  _spec_  and  _ref ,_ and with  `label`. The labels will be assigned to the  _label_  fields in  _object_.
```go
func UnmarshalSpec(dat []byte, current interface{}, spec *Struct, ref map[string]interface{}, label_values ...string) error 
//
// spec: describe how the interface fields are interprested
// ref: a reference map to map class names in spec, to objects of empty value.
// e.g.
// ref := map[string]interface{}{"cirle": new(Circle), "geo": new(Geo)}
```

In the following example, we decode data to  _child_  of type  _Nested of nested_, which contains multiple  _interfaces_  and  _maps_,

```go
package main

import (
    "fmt"

    "github.com/genelet/horizon/utils"
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
	Age   int  `json:"age" hcl:"age"`
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
        name = "quare shape"
        shape {
            sx = 5
            sy = 6
        }
    }
}
`
    spec, err := utils.NewStruct("child", map[string]interface{}{
        "Brand": map[string][2]interface{}{
            "abc1":[2]interface{}{"toy", map[string]interface{}{
                "Geo": [2]interface{}{
                    "geo", map[string]interface{}{"Shape": "circle"}}}},
            "def2":[2]interface{}{"toy", map[string]interface{}{
                "Geo": [2]interface{}{
                    "geo", map[string]interface{}{"Shape": "square"}}}},
        },
    })
    ref := map[string]interface{}{"toy": &toy{}, "geo": &geo{}, "circle": &circle{}, "square": &square{}}

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
  
the program outputs:  
  
```bash
    5  
    &main.toy{Geo:main.geo{Name:"medium shape", Shape:(*main.circle)(0xc000018650)}, ToyName:"roblox", Price:99.9}  
    &main.circle{Radius:1.234}  
    &main.toy{Geo:main.geo{Name:"quare shape", Shape:(*main.square)(0xc000018890)}, ToyName:"minecraft", Price:9.9}  
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

<br>

# Chapter 3. Conversion among Data Formats HCL, JSON and YAML
<br>

## 3.1 Introduction

Hashicorp Configuration Language ([HCL](https://github.com/hashicorp/hcl)) is a user-friendly data format for structured configuration. It combines parameters and declarative logic in a way that is easily understood by both humans and machines. HCL is integral to Hashicorp’s cloud infrastructure automation tools, such as  `Terraform`  and  `Nomad`. With its robust support for expression syntax, HCL has the potential to serve as a general data format with programming capabilities, making it suitable for use in no-code platforms.

However, in many scenarios, we still need to use popular data formats like JSON and YAML alongside HCL. For instance, Hashicorp products use JSON for data communication via REST APIs, while Docker or Kubernetes management in  `Terraform`  requires YAML.

## 3.2 Question

An intriguing question arises: Is it possible to convert HCL to JSON or YAML, and vice versa? Could we use HCL as the universal configuration language in projects and generate YAML or JSON with CLI or within  `Terraform`  on the fly?

Unfortunately, the answer is generally no. The expressive power of HCL surpasses that of JSON and YAML. In particular, HCL uses array key (i.e. labels) to express maps, while JSON and YAML use single maps. Most importantly, HCL allows variables and logic expressions, while JSON and YAML are purely data declarative. Therefore, some features in HCL can never be accurately represented in JSON.

However, in cases where we don’t care about map orders, and there are no variables or logical expressions, but only generic maps, lists, and scalars, then the answer is yes. This type of HCL can be accurately converted to JSON, and vice versa.

> There is a practical advantage of HCL over YAML: HCL is very readable and less prone to errors, while YAML is sensitive to markers like white-space. One can write a configuration in HCL and let a program handle conversion.


## 3.3 The Package

`horizon`  is a GO package to marshal and unmarshal dynamic JSON and HCL contents with interface types. It has a  `convert`  library for conversions among different data formats.

Technically, a JSON or YAML string can be unmarshalled into an anonymous map of  _map[string]interface{}_. For seamless conversion, `horizon` has internally implemented methods to unmarshal any HCL string into an anonymous map, and marshal an anonymous map into a properly formatted HCL string.

The following functions in  `horizon/convert`  can be used for conversion:

-   hcl to json:  _HCLToJSON(raw []byte) ([]byte, error)_
-   hcl to yaml:  _HCLToYAML(raw []byte) ([]byte, error)_
-   json to hcl:  _JSONToHCL(raw []byte) ([]byte, error)_
-   json to yaml:  _JSONToYAML(raw []byte) ([]byte, error)_
-   yaml to hcl:  _YAMLToHCL(raw []byte) ([]byte, error)_
-   yaml to json:  _YAMLToJSON(raw []byte) ([]byte, error)_

If you start with HCL, make sure it contains only primitive data types of maps, lists and scalars.

> In HCL, square brackets are lists and curly brackets are maps. Use  **equal sign  _=_**  and  **comma**  to separate values for  **list** assignment. But no equal sign nor comma for map.

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

> Note that HCL is enclosed internally in curly bracket. But the top-level curly bracket should be removed, so it can be accepted by  [the HCL parser](https://pkg.go.dev/github.com/hashicorp/hcl/v2/hclsyntax).

Run the program to get YAML:
```bash
    $ go run x.go  
    
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
## 3.4 The CLI

In directory  `cmd`, there is a CLI program  `convert.go`. Its usage is

```bash      
    $ go run convert.go  
    
    convert [options] <filename>  
      -from string  
         from format (default "hcl")  
      -to string  
         to format (default "yaml")
 ```   

This is a HCL:

```bash
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
    $ go run convert.go -to json the_above.hcl   
    
    {"services":{"api":{"depends_on":["db"],"environment":{"CONFIG_FILE":"/config/config.json"},"image":"hashicorpdemoapp/product-api:v0.0.22","ports":["19090:9090"],"volumes":["./conf.json:/config/config.json"]},"db":{"environment":{"POSTGRES_DB":"products","POSTGRES_PASSWORD":"password","POSTGRES_USER":"postgres"},"image":"hashicorpdemoapp/product-api-db:v0.0.22","ports":["15432:5432"]}},"version":"3.7"}
```
Convert it to YAML:
```bash
    $ go run convert.go the_above.hcl  
    
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
We see that HCL’s syntax is cleaner, more readable, and less error-prone compared to JSON and YAML.

## 3.5 Summary

HCL is a novel data format that offers advantages over JSON and YAML. In this article, we have demonstrated how to convert data among these three formats.
