package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/genelet/horizon/convert"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [options] <filename>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nSupported formats: json, yaml, hcl\n\n")
	flag.PrintDefaults()
	os.Exit(1)
}

// conversionFunc represents a format conversion function
type conversionFunc func([]byte) ([]byte, error)

// conversions maps "from->to" format pairs to their conversion functions
var conversions = map[string]conversionFunc{
	"json->yaml": convert.JSONToYAML,
	"json->hcl":  convert.JSONToHCL,
	"yaml->json": convert.YAMLToJSON,
	"yaml->hcl":  convert.YAMLToHCL,
	"hcl->json":  convert.HCLToJSON,
	"hcl->yaml":  convert.HCLToYAML,
}

func main() {
	var from string
	var to string
	flag.StringVar(&from, "from", "json", "from format")
	flag.StringVar(&to, "to", "hcl", "to format")
	flag.Parse()

	if from == to {
		fmt.Fprintf(os.Stderr, "error: from and to format are the same\n")
		os.Exit(1)
	}

	filename := flag.Arg(0)
	if filename == "" {
		usage()
	}

	raw, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Look up conversion function
	conversionKey := from + "->" + to
	convertFunc, ok := conversions[conversionKey]
	if !ok {
		fmt.Fprintf(os.Stderr, "error: unsupported conversion from %s to %s\n", from, to)
		os.Exit(1)
	}

	// Perform conversion
	raw, err = convertFunc(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", raw)
}
