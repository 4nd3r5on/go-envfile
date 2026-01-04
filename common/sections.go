package common

import (
	"regexp"
	"strings"
)

var (
	SectionStartRe = regexp.MustCompile(`^# \[SECTION:\s*([^]]+)\](.*)$`)
	SectionEndRe   = regexp.MustCompile(`^# \[SECTION_END:\s*([^]]+)\](.*)$`)
)

// MatchedSectionData holds information about a section marker.
type MatchedSectionData struct {
	Name    string
	Comment string
}

// MakeSectionStart creates a section start marker with optional inline comment.
// Example: "# [SECTION: my_section] some comment".
func MakeSectionStart(
	name string,
	comment string,
	ensureNewLine bool,
) string {
	if name == "" {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("# [SECTION: ")
	sb.WriteString(name)
	sb.WriteByte(']')

	if comment != "" {
		sb.WriteByte(' ')
		sb.WriteString(comment)
	}

	if ensureNewLine {
		sb.WriteByte('\n')
	}

	return sb.String()
}

// MakeSectionEnd creates a section end marker with optional inline comment.
// Example: "# [SECTION_END: my_section] reason".
func MakeSectionEnd(
	name string,
	comment string,
	ensureNewLine bool,
) string {
	if name == "" {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("# [SECTION_END: ")
	sb.WriteString(name)
	sb.WriteByte(']')

	if comment != "" {
		sb.WriteByte(' ')
		sb.WriteString(comment)
	}

	if ensureNewLine {
		sb.WriteByte('\n')
	}

	return sb.String()
}

// MatchSectionStart checks if a line is a section start marker
// Returns true and section data if matched, false otherwise.
func MatchSectionStart(line string) (bool, MatchedSectionData) {
	matches := SectionStartRe.FindStringSubmatch(line)
	if matches == nil {
		return false, MatchedSectionData{}
	}

	return true, MatchedSectionData{
		Name:    strings.TrimSpace(matches[1]),
		Comment: strings.TrimSpace(matches[2]),
	}
}

// MatchSectionEnd checks if a line is a section end marker
// Returns true and section data if matched, false otherwise.
func MatchSectionEnd(line string) (bool, MatchedSectionData) {
	matches := SectionEndRe.FindStringSubmatch(line)
	if matches == nil {
		return false, MatchedSectionData{}
	}

	return true, MatchedSectionData{
		Name:    strings.TrimSpace(matches[1]),
		Comment: strings.TrimSpace(matches[2]),
	}
}
