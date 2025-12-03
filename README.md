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

## Documentation

For detailed documentation and tutorials, please refer to [DOCUMENT.md](DOCUMENT.md).

- Chapter 1: [Marshal GO Object into HCL](DOCUMENT.md#chapter-1-marshal-go-object-into-hcl)
- Chapter 2: [Unmarshal HCL Data to GO Object](DOCUMENT.md#chapter-2-unmarshal-hcl-data-to-go-object)
- Chapter 3: [Conversion among Data Formats HCL, JSON and YAML](DOCUMENT.md#chapter-3-conversion-among-data-formats-hcl-json-and-yaml)

## Usage

### Library Usage (dethcl)

The `dethcl` package provides `Marshal` and `Unmarshal` functions similar to `encoding/json`.

```go
package main

import (
    "fmt"
    "github.com/genelet/horizon/dethcl"
)

type Config struct {
    Name    string   `hcl:"name"`
    Enabled bool     `hcl:"enabled,optional"`
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

## License

See [LICENSE](LICENSE) file.
