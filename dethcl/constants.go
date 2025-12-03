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

	// markerNoBrackets is an internal sentinel value used during array encoding.
	// When passed as keyname[0] to marshalLevel, it signals that:
	// 1. Zero values should be encoded (not skipped)
	// 2. Map-style brackets should not wrap the output
	//
	// This marker is intentionally cryptic to avoid collision with user data.
	// It's only used internally and never appears in output.
	markerNoBrackets = "__DETHCL_NO_BRACKETS_MARKER__"
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
)
