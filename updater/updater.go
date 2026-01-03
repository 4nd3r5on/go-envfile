package updater

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/4nd3r5on/go-envfile/common"
)

type Update struct {
	Key     string
	Value   string
	Section string // empty string for no section

	// If variable already exists -- won't move section for this specific variable
	IgnoreSection bool

	Prefix string // for "export " before key for example
	// Works only for adding variables
	// If variable existed before -- keeping existing suffix
	InlineComment string
}

// FromStream processes a parser stream and generates patches based on the provided updates.
// It returns a map of patches keyed by line index to be applied to the original content.
func FromStream(
	s common.ParserStream,
	updates []Update,
	options ...Option,
) (map[int64]common.Patch, error) {
	cfg := &Config{
		Logger:               slog.Default(),
		EnsureNewLine:        true,
		Mode:                 ModeReplace | ModeAdd | ModeMoveSection,
		SectionStartComments: make(map[string]string),
		SectionEndComments:   make(map[string]string),
	}
	for _, option := range options {
		option(cfg)
	}

	updateMap := make(map[string]Update, len(updates))
	for _, update := range updates {
		if _, exists := updateMap[update.Key]; exists {
			return nil, fmt.Errorf("duplicate update for key %q: each key must appear only once in updates", update.Key)
		}
		updateMap[update.Key] = update
		cfg.Logger.Debug("registered update", "key", update.Key, "section", update.Section)
	}

	var (
		lineIdx              int64
		currentSection       string
		lastVarLineInSection int64
		sectionsLastVarLine  = make(map[string]int64)
		addToSection         = make(map[string]string)
		patchMap             = make(map[int64]common.Patch)
	)

	cfg.Logger.Info("starting stream processing", "total_updates", len(updates))

	// Parse the stream
	for {
		lineIdx = s.GetLineIdx()
		parsedLine, err := s.Next()

		if err == io.EOF {
			cfg.Logger.Debug("reached end of stream", "final_line", lineIdx)
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse line %d: %w", lineIdx, err)
		}

		switch parsedLine.Type {
		case common.LineTypeSectionStart:
			if parsedLine.SectionData == nil {
				return nil, fmt.Errorf("line %d: section start detected but SectionData is nil", lineIdx)
			}
			currentSection = parsedLine.SectionData.Name
			sectionsLastVarLine[currentSection] = lineIdx
			cfg.Logger.Debug("entered section", "section", currentSection, "line", lineIdx)

		case common.LineTypeSectionEnd:
			cfg.Logger.Debug("section end", "section", currentSection, "line", lineIdx)

		case common.LineTypeVar:
			lastVarLineInSection = lineIdx

			if parsedLine.Variable == nil {
				return nil, fmt.Errorf("line %d: variable line detected but Variable is nil", lineIdx)
			}

			varKey := parsedLine.Variable.Key
			varLinesBuffer := []common.ParsedLine{parsedLine}

			// Check if this variable needs updating
			varUpdate, shouldUpdate := updateMap[varKey]
			if !shouldUpdate {
				cfg.Logger.Debug("skipping variable (no update)", "key", varKey, "line", lineIdx)
				continue
			}

			cfg.Logger.Debug("processing variable update", "key", varKey, "line", lineIdx, "terminated", parsedLine.Variable.IsTerminated)

			// Handle multi-line values
			if !parsedLine.Variable.IsTerminated {
				parsedValLines, err := parseUntilTerminated(s, lineIdx, varKey, cfg.Logger)
				if err != nil {
					return nil, fmt.Errorf("line %d: failed to parse multi-line value for %q: %w", lineIdx, varKey, err)
				}
				varLinesBuffer = append(varLinesBuffer, parsedValLines...)
				lastVarLineInSection += int64(len(parsedValLines))
				cfg.Logger.Debug("parsed multi-line value", "key", varKey, "total_lines", len(varLinesBuffer))
			}

			// Process the update
			updateBlock := processVarUpdate(lineIdx, varUpdate, varLinesBuffer, cfg.EnsureNewLine, cfg.Logger)

			// Apply patches
			for _, patch := range updateBlock.Patches {
				if _, exists := patchMap[patch.LineIdx]; exists {
					cfg.Logger.Warn("overwriting existing patch", "line", patch.LineIdx, "key", varKey)
				}
				patchMap[patch.LineIdx] = patch
			}

			// Track content to add to section
			if updateBlock.AddVariable != nil && updateBlock.AddVariable.Content != "" {
				addToSection[updateBlock.AddVariable.Section] += updateBlock.AddVariable.Content
			}

			// Mark update as processed
			delete(updateMap, varKey)
			cfg.Logger.Debug("applied variable update", "key", varKey, "patches", len(updateBlock.Patches))
		}

		// Update section tracking
		if parsedLine.Type == common.LineTypeVar {
			sectionsLastVarLine[currentSection] = lastVarLineInSection
		}
	}

	// Process remaining updates (new variables)
	if len(updateMap) > 0 {
		cfg.Logger.Info("processing new variables", "count", len(updateMap))
		for key, update := range updateMap {
			newVar := FormatVar(update, nil, cfg.EnsureNewLine)
			addToSection[update.Section] += newVar
			cfg.Logger.Debug("formatted new variable", "key", key, "section", update.Section)
		}
	}

	// Build content to add to sections and file end
	var addToFileEnd string

	for sectionName, addContent := range addToSection {
		if addContent == "" {
			continue
		}

		// Empty section means add to file end (no section)
		if sectionName == "" {
			addToFileEnd += addContent
			cfg.Logger.Debug("queued content for file end", "length", len(addContent))
			continue
		}

		lastVarLine, sectionExists := sectionsLastVarLine[sectionName]

		if !sectionExists {
			// Create new section at end of file
			cfg.Logger.Debug("creating new section", "section", sectionName)

			startComment := cfg.SectionStartComments[sectionName]
			if startComment == "" {
				startComment = cfg.SectionStartComments[""]
			}

			endComment := cfg.SectionEndComments[sectionName]
			if endComment == "" {
				endComment = cfg.SectionEndComments[""]
			}

			sectionStart := common.MakeSectionStart(sectionName, startComment, cfg.EnsureNewLine)
			sectionEnd := common.MakeSectionEnd(sectionName, endComment, cfg.EnsureNewLine)
			addToFileEnd += sectionStart + addContent + sectionEnd + "\n"
		} else {
			// Insert after last variable in existing section
			cfg.Logger.Debug("inserting into existing section", "section", sectionName, "after_line", lastVarLine)

			existingPatch, hasPatch := patchMap[lastVarLine]
			if hasPatch {
				existingPatch.InsertAfter += addContent
				patchMap[lastVarLine] = existingPatch
			} else {
				patchMap[lastVarLine] = common.Patch{
					LineIdx:           lastVarLine,
					ShouldInsertAfter: true,
					InsertAfter:       addContent,
				}
			}
		}
	}

	// Add file end content
	if addToFileEnd != "" {
		lineIdx := max(0, lineIdx-1)

		cfg.Logger.Debug("adding content to file end", "line", lineIdx, "length", len(addToFileEnd))
		existingPatch, hasPatch := patchMap[lineIdx]
		if hasPatch {
			existingPatch.InsertAfter += addToFileEnd
			patchMap[lineIdx] = existingPatch
		} else {
			patchMap[lineIdx] = common.Patch{
				LineIdx:           lineIdx,
				ShouldInsertAfter: true,
				InsertAfter:       addToFileEnd,
			}
		}
	}

	cfg.Logger.Info("stream processing complete", "total_patches", len(patchMap), "unprocessed_updates", len(updateMap))

	return patchMap, nil
}

// parseUntilTerminated reads continuation lines until the value is terminated
func parseUntilTerminated(s common.ParserStream, startLine int64, varKey string, logger *slog.Logger) ([]common.ParsedLine, error) {
	buf := make([]common.ParsedLine, 0)

	for {
		parsedLine, err := s.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("unexpected end of input: multi-line value for %q starting at line %d was not terminated", varKey, startLine)
		}
		if err != nil {
			return nil, fmt.Errorf("error reading continuation line: %w", err)
		}

		if parsedLine.VariableValPart == nil {
			return nil, fmt.Errorf("continuation line missing value part (line type: %d)", parsedLine.Type)
		}

		// Validate continuity: line number should match buffer position
		// +1 because we're counting from the definition line (not in buffer)
		expectedLineNum := len(buf) + 1
		if parsedLine.UnterminatedValueLines != expectedLineNum {
			return nil, fmt.Errorf("multi-line value continuity error: expected continuation line %d, got line %d", expectedLineNum, parsedLine.UnterminatedValueLines)
		}

		buf = append(buf, parsedLine)

		if parsedLine.VariableValPart.IsTerminated {
			logger.Debug("multi-line value terminated", "key", varKey, "total_continuation_lines", len(buf))
			return buf, nil
		}
	}
}
