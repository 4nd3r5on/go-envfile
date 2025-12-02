package parser

import "regexp"

var (
	SectionStartRe = regexp.MustCompile(`(?m)^# \[SECTION:\s*([^\]]+)\].*$`)
	SectionEndRe   = regexp.MustCompile(`(?m)^# \[SECTION_END:\s*([^\]]+)\].*$`)
)
