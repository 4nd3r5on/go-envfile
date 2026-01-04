package updater

import (
	"fmt"
	"strings"

	"github.com/4nd3r5on/go-envfile/common"
)

// HandleEOF processes end-of-file state and returns all accumulated patches.
// It validates state, processes pending updates, and organizes content insertion.
func (u *Updater) HandleEOF(lineIdx int64) (map[int64]common.Patch, error) {
	u.Logger.Debug("reached end of stream", "final_line", lineIdx)

	if u.varState != nil && !u.varState.IsTerminated {
		return nil, fmt.Errorf(
			"EOF with unterminated variable %s on line %d",
			u.varState.Key,
			u.varState.DefinitionLine,
		)
	}

	u.processNewVariables()
	u.distributeContentToSections(lineIdx)

	return u.patchMap, nil
}

// processNewVariables formats and stages all pending variable updates.
func (u *Updater) processNewVariables() {
	if len(u.updateMap) == 0 {
		return
	}

	u.Logger.Info("processing new variables", "count", len(u.updateMap))

	for key, update := range u.updateMap {
		formattedVar := FormatVar(update, nil, true, u.DefaultQuote)
		u.addToSection[update.Section] += formattedVar
		u.Logger.Debug("formatted new variable", "key", key, "section", update.Section)
	}
}

// distributeContentToSections inserts staged content into appropriate sections.
func (u *Updater) distributeContentToSections(eofLine int64) {
	var contentForNewSections string

	for sectionName, content := range u.addToSection {
		if content == "" {
			continue
		}

		lastVarLine, exists := u.sectionsLastVarLine[sectionName]
		if exists {
			u.insertIntoExistingSection(sectionName, lastVarLine, content)
		} else {
			contentForNewSections += u.createSection(sectionName, content)
		}
	}

	if contentForNewSections != "" {
		u.appendToFileEnd(contentForNewSections, eofLine)
	}
}

// insertIntoExistingSection adds content after the last variable in a section.
func (u *Updater) insertIntoExistingSection(sectionName string, lastVarLine int64, content string) {
	u.Logger.Debug("inserting into existing section",
		"section", sectionName,
		"after_line", lastVarLine)

	patch := u.getOrCreatePatch(lastVarLine)
	patch.ShouldInsertAfter = true
	patch.InsertAfter += content
	u.patchMap[lastVarLine] = patch
}

// appendToFileEnd adds content to the end of the file.
func (u *Updater) appendToFileEnd(content string, eofLine int64) {
	// Use the last valid line index (subtract 1 to account for EOF marker)
	lineIdx := max(0, eofLine-1)

	u.Logger.Debug("adding content to file end",
		"line", lineIdx,
		"length", len(content))

	patch := u.getOrCreatePatch(lineIdx)
	patch.ShouldInsertAfter = true
	patch.InsertAfter += content
	u.patchMap[lineIdx] = patch
}

// getOrCreatePatch retrieves an existing patch or creates a new one.
func (u *Updater) getOrCreatePatch(lineIdx int64) common.Patch {
	if patch, exists := u.patchMap[lineIdx]; exists {
		return patch
	}
	return common.Patch{LineIdx: lineIdx}
}

// createSection builds a complete section with comments and content.
func (u *Updater) createSection(name, content string) string {
	startComment := getSectionComment(name, u.SectionStartComments)
	endComment := getSectionComment(name, u.SectionEndComments)

	sectionStart := common.MakeSectionStart(name, startComment, false)
	sectionEnd := common.MakeSectionEnd(name, endComment, false)

	var builder strings.Builder
	builder.WriteString(sectionStart)
	builder.WriteByte('\n')
	builder.WriteString(content)
	builder.WriteByte('\n')
	builder.WriteString(sectionEnd)
	builder.WriteByte('\n')

	return builder.String()
}

// getSectionComment retrieves the comment for a section, falling back to default.
func getSectionComment(name string, commentMap map[string]string) string {
	if comment, exists := commentMap[name]; exists && comment != "" {
		return comment
	}
	return commentMap[""]
}
