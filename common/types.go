//revive:disable:var-naming
package common

//revive:enable:var-naming

type Span[T comparable] struct {
	Start T
	End   T
}

// ByteSpan = [start,end] byte offsets.
type ByteSpan Span[int64]

// Interfaces

type Parser interface {
	ParseLine(line string) (ParsedLine, error)
}

type ParserStream interface {
	GetLineIdx() int64
	Next() (ParsedLine, error) // parses until io.EOF
}

// Parser

type LineType int

const (
	LineTypeRaw     LineType = iota
	LineTypeComment          // Only if line starts with #
	LineTypeVar
	LineTypeVal // If value wasn't terminated on the previous line
	LineTypeSectionStart
	LineTypeSectionEnd
)

// VariableData contains parsed variable information.
type VariableData struct {
	Key          string
	Value        string
	Prefix       string // Everything before the value (export, whitespace, key, =, etc)
	Suffix       string // Everything after the value (whitespace, comments)
	IsTerminated bool
	IsQuoted     bool
	Quote        byte
}

type VariableValPartData struct {
	Value        string // What value did variable have
	Suffix       string // Everything after the value (whitespace, comments)
	IsTerminated bool   // If variable was terminated on that line
	Quote        byte
}

type SectionData struct {
	Variables map[string]struct{} // key value quick lookup for sections
	Name      string
}

// ParsedLine represents a parsed line from .env file.
type ParsedLine struct {
	Type    LineType
	RawLine string

	Variable        *VariableData
	VariableValPart *VariableValPartData
	SectionData     *SectionData // if line is within a section

	SectionStartEndInlineComment string

	// if value wasn't terminated > 0. 0 if nothing to terminate or terminated the same line
	UnterminatedValueLines int
}

// Updater

// Patch represents changes that u need to put into the file
// Warning: Insert And InsertAfter don't automatically add new lines.
type Patch struct {
	LineIdx      int64 // target line
	ShouldInsert bool
	Insert       string // insert before the target line

	ShouldInsertAfter bool // same as should insert, but next line
	InsertAfter       string

	RemoveLine bool // removes the target line
}
