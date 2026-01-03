package updater

import (
	"log/slog"
	"strings"

	"github.com/4nd3r5on/go-envfile/common"
)

type AddVariable struct {
	Section string
	Content string
}

// FormatVar creates a formatted variable line from an update and optional original data.
// Takes value from the update while preserving formatting from the original (like comment and prefix if exists).
func FormatVar(update Update, orig *common.VariableData, ensureNewLine bool) string {
	var (
		prefix string
		suffix string
		value  string
	)

	if orig != nil {
		prefix = orig.Prefix
		suffix = orig.Suffix
	} else {
		if update.Prefix != "" {
			prefix = update.Prefix + update.Key + "="
		} else {
			prefix = update.Key + "="
		}
	}

	// Determine quoting strategy
	var quote byte
	switch {
	case orig != nil && orig.IsQuoted:
		quote = orig.Quote
	case common.HasSpaceChars(update.Value):
		quote = '"'
	}

	value = update.Value

	// Escape quotes if quoting is used
	if quote != 0 {
		escaped := make([]byte, 0, len(value))
		for i := 0; i < len(value); i++ {
			switch value[i] {
			case quote, '\\':
				escaped = append(escaped, '\\')
			}
			escaped = append(escaped, value[i])
		}
		value = string(escaped)
		value = string(quote) + value + string(quote)
	}

	// Add inline comment if suffix is empty/whitespace and comment is provided
	if common.IsEmptyStr(suffix) && update.InlineComment != "" {
		suffix = " # " + update.InlineComment
	}

	if ensureNewLine && (len(suffix) == 0 || suffix[len(suffix)-1] != '\n') {
		suffix += "\n"
	}

	return prefix + value + suffix
}

// UpdateBlock represents a complete update operation for a variable.
// Contains all patches needed to remove old lines and optionally insert new content.
type UpdateBlock struct {
	// Patches to apply to the file, ordered by line index
	Patches []common.Patch
	// If variable needs to be added in a different section
	AddVariable *AddVariable
}

// reconstructMultiLineValue reconstructs the complete value from multiple parsed lines.
// origLines contains the definition line and all continuation lines.
func reconstructMultiLineValue(origLines []common.ParsedLine) string {
	if len(origLines) == 0 {
		return ""
	}

	var valueParts []string

	// First line contains the initial value
	if origLines[0].Variable != nil {
		valueParts = append(valueParts, origLines[0].Variable.Value)
	}

	// Subsequent lines contain value continuation parts
	for i := 1; i < len(origLines); i++ {
		if origLines[i].VariableValPart != nil {
			valueParts = append(valueParts, origLines[i].VariableValPart.Value)
		}
	}

	// Join with newlines to preserve multi-line structure
	return strings.Join(valueParts, "\n")
}

// processVarUpdate creates an update block for updating a variable.
// origLines must contain at least one line (the variable definition line).
// First line defines the variable, subsequent lines are value continuation parts.
// Returns an UpdateBlock containing all necessary patches and optional move information.
func processVarUpdate(
	lineIdx int64,
	update Update,
	origLines []common.ParsedLine,
	ensureNewLine bool,
	logger *slog.Logger,
) UpdateBlock {
	if len(origLines) == 0 {
		logger.Error("processVarUpdate called with empty origLines", "key", update.Key)
		return UpdateBlock{
			Patches: []common.Patch{},
			AddVariable: &AddVariable{
				Section: update.Section,
				Content: FormatVar(update, nil, ensureNewLine),
			},
		}
	}

	definitionLine := origLines[0]
	if definitionLine.Variable == nil {
		logger.Error("definition line missing Variable data", "key", update.Key, "line", lineIdx)
		return UpdateBlock{
			Patches: []common.Patch{},
			AddVariable: &AddVariable{
				Section: update.Section,
				Content: FormatVar(update, nil, ensureNewLine),
			},
		}
	}

	// Reconstruct the original multi-line value to compare with update
	originalValue := reconstructMultiLineValue(origLines)

	// Determine current section
	var currentSection string
	if definitionLine.SectionData != nil {
		currentSection = definitionLine.SectionData.Name
	}

	// Check if value matches
	valCorrect := originalValue == update.Value

	// Check if section matches
	sectionCorrect := currentSection == update.Section

	// If IgnoreSection is set, treat section as correct regardless
	if update.IgnoreSection {
		sectionCorrect = true
	}

	logger.Debug("variable update analysis",
		"key", update.Key,
		"line", lineIdx,
		"value_correct", valCorrect,
		"section_correct", sectionCorrect,
		"current_section", currentSection,
		"target_section", update.Section,
		"multiline", len(origLines) > 1,
	)

	// Case 1: Both value and section are correct - no changes needed
	if valCorrect && sectionCorrect {
		logger.Debug("no changes needed for variable", "key", update.Key)
		return UpdateBlock{
			Patches: []common.Patch{},
		}
	}

	// Create patches to remove all lines of the variable (definition + continuation lines)
	patches := make([]common.Patch, len(origLines))
	for i := 0; i < len(origLines); i++ {
		patches[i] = common.Patch{
			LineIdx:    lineIdx + int64(i),
			RemoveLine: true,
		}
	}

	// Format the new variable content
	varContent := FormatVar(update, definitionLine.Variable, ensureNewLine)

	// Case 2: Value needs updating, but section is correct - update in place
	if !valCorrect && sectionCorrect {
		logger.Debug("updating variable in place", "key", update.Key, "line", lineIdx)
		// Insert new content before removing the first line
		patches[0].Insert = varContent
		patches[0].ShouldInsert = true

		return UpdateBlock{
			Patches: patches,
		}
	}

	// Case 3: Section needs changing (with or without value change)
	logger.Debug("moving variable to different section",
		"key", update.Key,
		"from_section", currentSection,
		"to_section", update.Section,
	)

	// Remove all lines from current location and mark for insertion in new section
	return UpdateBlock{
		Patches: patches,
		AddVariable: &AddVariable{
			Section: update.Section,
			Content: varContent,
		},
	}
}
