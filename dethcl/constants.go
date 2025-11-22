package dethcl

// HCL struct tag constants
const (
	// tagPrefixHCL is the prefix for HCL struct tags
	tagPrefixHCL = "hcl:\""

	// tagPrefixHCLLength is the length of "hcl:\""
	tagPrefixHCLLength = 5

	// tagModifierLabel indicates a field is an HCL label
	tagModifierLabel = "label"

	// tagModifierBlock indicates a field is an HCL block
	tagModifierBlock = "block"

	// tagModifierOptional indicates a field is optional
	tagModifierOptional = "optional"

	// tagIgnore indicates a field should be ignored
	tagIgnore = "-"

	// tagIgnoreSuffix indicates a field with comma-dash suffix should be ignored
	tagIgnoreSuffix = ",-"

	// markerNoBrackets is a special marker used during encoding to indicate
	// that map brackets should not be used for array item encoding.
	// This is used internally when encoding array elements within a map structure.
	markerNoBrackets = "NMRBRCKTNDTRMND"
)

// File extension constants
const (
	// hclFileExtension is the standard HCL file extension
	hclFileExtension = ".hcl"
)

// Parsing and encoding constants
const (
	// tempAttributeName is the temporary attribute name used when parsing expressions
	// that need to be wrapped in an HCL attribute context
	tempAttributeName = "x"

	// hclIndentSpaces is the number of spaces used for each indentation level in HCL output
	hclIndentSpaces = 2

	// emptyArrayHCL is the HCL representation of an empty array
	emptyArrayHCL = "[]"

	// emptyObjectHCL is the HCL representation of an empty object
	emptyObjectHCL = "{}"
)
