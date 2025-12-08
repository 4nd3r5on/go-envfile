package parser

import (
	"bufio"
	"errors"
	"regexp"
	"strings"
	"unicode"

	"github.com/4nd3r5on/go-envfile/common"
)

var (
	SectionStartRe = regexp.MustCompile(`^# \[SECTION:\s*([^]]+)\](.*)$`)
	SectionEndRe   = regexp.MustCompile(`^# \[SECTION_END:\s*([^]]+)\](.*)$`)
)

type ValStartEnd struct {
	Val   string
	Start int
	End   int
}

// Parse everything from a scanner to an array or parsed lines
// useful when reading files
func Parse(p Parser, s *bufio.Scanner) ([]common.ParsedLine, error) {
	lines := make([]common.ParsedLine, 0)

	for s.Scan() {
		line, err := p.ParseLine(s.Text())
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}

	return lines, nil
}

// SkipSpaces returns the first index after spaces (starts skipping from pos)
// Returns len(line) if all remaining characters are spaces
func SkipSpaces(line string, pos int) int {
	for i := pos; i < len(line); i++ {
		if !unicode.IsSpace(rune(line[i])) {
			return i
		}
	}
	return len(line)
}

// UntilSpace returns the index of the first space starting from pos
// Returns len(line) if no space is found
func UntilSpace(line string, pos int) int {
	for i := pos; i < len(line); i++ {
		if unicode.IsSpace(rune(line[i])) {
			return i
		}
	}
	return len(line)
}

// SkipSpacesBack returns the last index (moving left from pos) that is NOT a space
// Returns -1 if all characters from 0 to pos are spaces
func SkipSpacesBack(line string, pos int) int {
	for i := pos; i >= 0; i-- {
		if !unicode.IsSpace(rune(line[i])) {
			return i
		}
	}
	return -1
}

// UntilSpaceBack returns the index of the first space encountered moving left from pos
// Returns -1 if no space is found
func UntilSpaceBack(line string, pos int) int {
	for i := pos; i >= 0; i-- {
		if unicode.IsSpace(rune(line[i])) {
			return i
		}
	}
	return -1
}

// IsLineComment checks if a line is a comment (starts with # after optional spaces)
func IsLineComment(line string) bool {
	if len(line) == 0 {
		return false
	}
	lineStartsAt := SkipSpaces(line, 0)
	if lineStartsAt >= len(line) {
		return false
	}
	return line[lineStartsAt] == '#'
}

func DetectLineType(line string) common.LineType {
	if len(line) == 0 {
		return common.LineTypeRaw
	}
	lineStartsAt := SkipSpaces(line, 0)
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
func ExtractKey(line string, equalIdx int) (ValStartEnd, error) {
	if equalIdx == 0 {
		return ValStartEnd{}, errors.New("no key: equals sign at start of line")
	}

	// Find last non-space before '='
	keyEnd := SkipSpacesBack(line, equalIdx-1)
	if keyEnd == -1 {
		return ValStartEnd{}, errors.New("no key: only spaces before equals sign")
	}

	// Find start of a key
	keyStart := UntilSpaceBack(line, keyEnd) + 1

	return ValStartEnd{
		Val:   line[keyStart : keyEnd+1],
		Start: keyStart,
		End:   keyEnd,
	}, nil
}

// ExtractValue extracts the value from line given the position of '='
// Returns value string, whether it's terminated, and any error
func ExtractValue(line string, equalIdx int) (data ValStartEnd, isTerminated bool, terminator byte, err error) {
	// Find value start
	valStart := SkipSpaces(line, equalIdx+1)
	if valStart >= len(line) {
		return ValStartEnd{}, false, byte(0), errors.New("no value: nothing after equals sign")
	}

	// Check if quoted
	if line[valStart] == '"' || line[valStart] == '\'' {
		return extractQuotedValue(line, valStart)
	}

	// Unquoted value
	return extractUnquotedValue(line, valStart), true, byte(0), nil
}

func findTerminator(line string, pos int, terminator byte) int {
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
func extractQuotedValue(line string, pos int) (data ValStartEnd, isTerminated bool, terminator byte, err error) {
	quote := line[pos]
	terminatorPos := findTerminator(line, pos, quote)
	if terminatorPos < 0 {
		// Unterminated
		return ValStartEnd{
			Val:   line[pos:],
			Start: pos,
			End:   len(line) - 1,
		}, false, quote, nil
	}
	return ValStartEnd{
		Val:   line[pos : terminatorPos+1],
		Start: pos,
		End:   terminatorPos,
	}, true, byte(0), nil
}

// extractUnquotedValue extracts an unquoted value starting at pos
func extractUnquotedValue(line string, pos int) ValStartEnd {
	valEnd := UntilSpace(line, pos)
	return ValStartEnd{
		Val:   line[pos:valEnd],
		Start: pos,
		End:   valEnd - 1,
	}
}

// ParseVariable parses a line for a variable assignment (KEY=VALUE)
// Returns isVariable, key, value, isTerminated, and error
func ParseVariable(line string) (key, val ValStartEnd, isTerminated bool, terminator byte, err error) {
	// Check if comment

	// Find equals sign
	equalIdx := strings.IndexByte(line, '=')
	if equalIdx == -1 {
		return ValStartEnd{}, ValStartEnd{}, false, byte(0), errors.New("no equals sign found, no variable declaration")
	}

	// Extract key
	key, err = ExtractKey(line, equalIdx)
	if err != nil {
		return ValStartEnd{}, ValStartEnd{}, false, byte(0), err
	}

	// Extract value
	val, isTerminated, terminator, err = ExtractValue(line, equalIdx)
	if err != nil {
		return ValStartEnd{}, ValStartEnd{}, false, byte(0), err
	}

	return key, val, isTerminated, terminator, nil
}

// MatchSectionStart checks if a line is a section start marker and extracts the name and comment.
// Returns:
//   - isSectionStart: true if the line matches the section start pattern
//   - name: the section name (trimmed of whitespace)
//   - comment: any text after the closing bracket (trimmed of leading whitespace)
func MatchSectionStart(line string) (isSectionStart bool, name, comment string) {
	matches := SectionStartRe.FindStringSubmatch(line)
	if matches == nil {
		return false, "", ""
	}

	name = strings.TrimSpace(matches[1])
	comment = strings.TrimSpace(matches[2])
	return true, name, comment
}

// MatchSectionEnd checks if a line is a section end marker and extracts the name and comment.
// Returns:
//   - isSectionEnd: true if the line matches the section end pattern
//   - name: the section name (trimmed of whitespace)
//   - comment: any text after the closing bracket (trimmed of leading whitespace)
func MatchSectionEnd(line string) (isSectionEnd bool, name, comment string) {
	matches := SectionEndRe.FindStringSubmatch(line)
	if matches == nil {
		return false, "", ""
	}

	name = strings.TrimSpace(matches[1])
	comment = strings.TrimSpace(matches[2])
	return true, name, comment
}
