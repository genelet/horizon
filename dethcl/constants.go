package dethcl

// HCL struct tag constants
const (
	// TagPrefixHCL is the prefix for HCL struct tags
	TagPrefixHCL = "hcl:\""

	// TagPrefixHCLLength is the length of "hcl:\""
	TagPrefixHCLLength = 5

	// TagModifierLabel indicates a field is an HCL label
	TagModifierLabel = "label"

	// TagModifierBlock indicates a field is an HCL block
	TagModifierBlock = "block"

	// TagModifierOptional indicates a field is optional
	TagModifierOptional = "optional"

	// TagIgnore indicates a field should be ignored
	TagIgnore = "-"

	// TagIgnoreSuffix indicates a field with comma-dash suffix should be ignored
	TagIgnoreSuffix = ",-"

	// MarkerNoBrackets is a special marker used during encoding to indicate
	// that map brackets should not be used for array item encoding.
	// This is used internally when encoding array elements within a map structure.
	MarkerNoBrackets = "NMRBRCKTNDTRMND"
)

// File extension constants
const (
	// HCLFileExtension is the standard HCL file extension
	HCLFileExtension = ".hcl"
)

// Parsing and encoding constants
const (
	// TempAttributeName is the temporary attribute name used when parsing expressions
	// that need to be wrapped in an HCL attribute context
	TempAttributeName = "x"

	// HCLIndentSpaces is the number of spaces used for each indentation level in HCL output
	HCLIndentSpaces = 2

	// EmptyArrayHCL is the HCL representation of an empty array
	EmptyArrayHCL = "[]"

	// EmptyObjectHCL is the HCL representation of an empty object
	EmptyObjectHCL = "{}"
)
