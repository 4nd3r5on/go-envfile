package parser

type ValueType int

const (
	ValueUnquoted ValueType = iota
	ValueSingleQuoted
	ValueDoubleQuoted
)

// ValueData holds information about an extracted value.
type ValueData struct {
	Raw          string    // The raw value including quotes if present
	Content      string    // The actual content (without quotes)
	Start        int       // Start position in the line
	End          int       // End position in the line
	Type         ValueType // Whether the value was quoted and with what
	IsTerminated bool      // Whether quotes were properly closed
}

// KeyData holds information about an extracted key.
type KeyData struct {
	Key   string
	Start int
	End   int
}

// VariableData holds information about a parsed variable.
type VariableData struct {
	Key   KeyData
	Value ValueData
}

func GetQuoteFromValType(t ValueType) (isQuoted bool, quote byte) {
	switch t {
	case ValueSingleQuoted:
		return true, '\''
	case ValueDoubleQuoted:
		return true, '"'
	default:
		return false, 0
	}
}
