package parser

import (
	"errors"
	"strings"

	"github.com/4nd3r5on/go-envfile/common"
)

// IsLineComment checks if a line is a comment (starts with # after optional spaces)
func IsLineComment(line string) bool {
	if len(line) == 0 {
		return false
	}
	lineStartsAt := common.SkipSpaces(line, 0)
	if lineStartsAt >= len(line) {
		return false
	}
	return line[lineStartsAt] == '#'
}

func DetectLineType(line string) common.LineType {
	if len(line) == 0 {
		return common.LineTypeRaw
	}
	lineStartsAt := common.SkipSpaces(line, 0)
	if lineStartsAt >= len(line) {
		return common.LineTypeRaw
	}
	if line[lineStartsAt] == '#' {
		return common.LineTypeComment
	}
	return common.LineTypeVar
}

// ExtractKey extracts the key from line given the position of '='
// Returns the key string or error if invalid
func ExtractKey(line string, equalIdx int) (KeyData, error) {
	if equalIdx == 0 {
		return KeyData{}, errors.New("no key: equals sign at start of line")
	}

	// Find last non-space before '='
	keyEnd := common.SkipSpacesBack(line, equalIdx-1)
	if keyEnd == -1 {
		return KeyData{}, errors.New("no key: only spaces before equals sign")
	}

	// Find start of a key
	keyStart := common.UntilSpaceBack(line, keyEnd) + 1

	return KeyData{
		Key:   line[keyStart : keyEnd+1],
		Start: keyStart,
		End:   keyEnd,
	}, nil
}

// ExtractValue extracts the value from line given the position of '='
// Returns structured value data and any error
func ExtractValue(line string, equalIdx int) (ValueData, error) {
	// Find value start
	valStart := common.SkipSpaces(line, equalIdx+1)
	if valStart >= len(line) {
		return ValueData{}, errors.New("no value: nothing after equals sign")
	}

	// Check if quoted
	char := line[valStart]
	if char == '"' || char == '\'' {
		return extractQuotedValue(line, valStart)
	}

	// Unquoted value
	return extractUnquotedValue(line, valStart), nil
}

// findTerminator finds the closing quote, accounting for escaping
func FindTerminator(line string, pos int, terminator byte) int {
	bsCount := 0 // count of consecutive backslashes
	for i := pos + 1; i < len(line); i++ {
		c := line[i]
		if c == '\\' {
			bsCount++
			continue
		}
		if c == terminator && bsCount%2 == 0 {
			return i
		}
		bsCount = 0
	}
	return -1
}

// extractQuotedValue extracts a quoted value starting at pos
func extractQuotedValue(line string, pos int) (ValueData, error) {
	quote := line[pos]
	terminatorPos := FindTerminator(line, pos, quote)

	valueType := ValueDoubleQuoted
	if quote == '\'' {
		valueType = ValueSingleQuoted
	}

	if terminatorPos < 0 {
		// Unterminated quote
		raw := line[pos:]
		content := raw[1:] // Remove opening quote
		return ValueData{
			Raw:          raw,
			Content:      content,
			Start:        pos,
			End:          len(line) - 1,
			Type:         valueType,
			IsTerminated: false,
		}, nil
	}

	// Properly terminated quote
	raw := line[pos : terminatorPos+1]
	content := raw[1 : len(raw)-1] // Remove both quotes
	return ValueData{
		Raw:          raw,
		Content:      content,
		Start:        pos,
		End:          terminatorPos,
		Type:         valueType,
		IsTerminated: true,
	}, nil
}

// extractUnquotedValue extracts an unquoted value starting at pos
func extractUnquotedValue(line string, pos int) ValueData {
	valEnd := common.UntilSpace(line, pos)
	raw := line[pos:valEnd]
	return ValueData{
		Raw:          raw,
		Content:      raw, // For unquoted, raw and content are the same
		Start:        pos,
		End:          valEnd - 1,
		Type:         ValueUnquoted,
		IsTerminated: true,
	}
}

// ParseVariable parses a line for a variable assignment (KEY=VALUE)
// Returns variable data and error
func ParseVariable(line string) (VariableData, error) {
	// Find equals sign
	equalIdx := strings.IndexByte(line, '=')
	if equalIdx == -1 {
		return VariableData{}, errors.New("no equals sign found, no variable declaration")
	}

	// Extract key
	key, err := ExtractKey(line, equalIdx)
	if err != nil {
		return VariableData{}, err
	}

	// Extract value
	val, err := ExtractValue(line, equalIdx)
	if err != nil {
		return VariableData{}, err
	}

	return VariableData{
		Key:   key,
		Value: val,
	}, nil
}
