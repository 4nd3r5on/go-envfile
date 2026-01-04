package common

import "unicode"

// SkipSpaces returns the first index after spaces (starts skipping from pos)
// Returns len(line) if all remaining characters are spaces.
func SkipSpaces(line string, pos int) int {
	for i := pos; i < len(line); i++ {
		if !unicode.IsSpace(rune(line[i])) {
			return i
		}
	}

	return len(line)
}

// UntilSpace returns the index of the first space starting from pos
// Returns len(line) if no space is found.
func UntilSpace(line string, pos int) int {
	for i := pos; i < len(line); i++ {
		if unicode.IsSpace(rune(line[i])) {
			return i
		}
	}

	return len(line)
}

// SkipSpacesBack returns the last index (moving left from pos) that is NOT a space
// Returns -1 if all characters from 0 to pos are spaces.
func SkipSpacesBack(line string, pos int) int {
	for i := pos; i >= 0; i-- {
		if !unicode.IsSpace(rune(line[i])) {
			return i
		}
	}

	return -1
}

// UntilSpaceBack returns the index of the first space encountered moving left from pos
// Returns -1 if no space is found.
func UntilSpaceBack(line string, pos int) int {
	for i := pos; i >= 0; i-- {
		if unicode.IsSpace(rune(line[i])) {
			return i
		}
	}

	return -1
}

// IsEmptyStr checks if a string contains only whitespace characters.
func IsEmptyStr(str string) bool {
	for _, c := range str {
		if !unicode.IsSpace(c) {
			return false
		}
	}

	return true
}

// IsEmptyStr checks if a string contains any whitespace characters.
func HasSpaceChars(str string) bool {
	for _, c := range str {
		if unicode.IsSpace(c) {
			return true
		}
	}

	return false
}
