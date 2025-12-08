package common

// Interfaces

type Parser interface {
	ParseLine(line string) (ParsedLine, error)
}

type ParserStream interface {
	Next() (ParsedLine, error) // parses until io.EOF
}

type Updater interface {
	AddUpdates([]Update) error
	FromStream(ParserStream) (map[int64]Patch, error)
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

// VariableData contains parsed variable information
type VariableData struct {
	Key          string
	Value        string
	Prefix       string // Everything before the value (export, whitespace, key, =, etc)
	Suffix       string // Everything after the value (whitespace, comments)
	IsTerminated bool
}

type VariableValPartData struct {
	Value        string // What value did variable have
	Suffix       string // Everything after the value (whitespace, comments)
	IsTerminated bool   // If variable was terminated on that line
}

type SectionData struct {
	Name string
}

// ParsedLine represents a parsed line from .env file
type ParsedLine struct {
	Type    LineType
	RawLine string

	Variable        *VariableData
	VariableValPart *VariableValPartData
	SectionData     *SectionData // if line is within a section

	SectionStartEndInlineComment string

	UnterminatedValueLines int
}

// Updater

type Update struct {
	Key     string
	Value   string
	Section string

	Prefix        string // for "export " before key for example
	InlineComment string
}

// Patch represents changes that u need to put into the file
type Patch struct {
	LineIdx    int64  // target line
	Insert     string // insert before the target line
	RemoveLine bool   // removes the target line
}
