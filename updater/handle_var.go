package updater

import (
	"fmt"

	"github.com/4nd3r5on/go-envfile/common"
)

func (u *Updater) handleVar(lineIdx int64, parsedLine common.ParsedLine) error {
	if parsedLine.Variable == nil {
		return fmt.Errorf("line %d: variable line detected but Variable is nil", lineIdx)
	}

	u.sectionsLastVarLine[u.currentSection] = lineIdx
	u.varState = &VariableState{
		DefinitionLine: lineIdx,
		Key:            parsedLine.Variable.Key,
		IsTerminated:   parsedLine.Variable.IsTerminated,
		LinesBuf:       []common.ParsedLine{parsedLine},
	}
	u.Logger.Debug("found variable", "line", lineIdx, "key", u.varState.Key, "is_terminated", u.varState.IsTerminated)

	return u.patchVar()
}

func (u *Updater) handleValPart(lineIdx int64, parsedLine common.ParsedLine) error {
	if parsedLine.VariableValPart == nil {
		return fmt.Errorf("line %d: variable part line detected but VariableValPart is nil", lineIdx)
	}

	if u.varState == nil {
		return fmt.Errorf("line %d: variable part line detected but Updater.varState is nil (no var declaration? most likely a bug)", lineIdx)
	}

	u.sectionsLastVarLine[u.currentSection] = lineIdx
	u.varState.LinesBuf = append(u.varState.LinesBuf, parsedLine)
	u.varState.IsTerminated = parsedLine.VariableValPart.IsTerminated

	return u.patchVar()
}

func (u *Updater) patchVar() error {
	if u.varState == nil {
		return nil // nothing to patch
	}

	if !u.varState.IsTerminated {
		return nil // not terminated yet
	}

	varUpdate, shouldUpdate := u.updateMap[u.varState.Key]
	if !shouldUpdate {
		u.Logger.Debug("skipping variable (no update)", "key", u.varState.Key, "line", u.varState.DefinitionLine)

		u.varState = nil

		return nil
	}

	updateBlock := processVarUpdate(
		u.varState.DefinitionLine,
		varUpdate,
		u.varState.LinesBuf,
		u.EnsureNewLine,
		u.DefaultQuote,
		u.Logger,
	)

	// Apply patches
	for _, patch := range updateBlock.Patches {
		u.patchMap[patch.LineIdx] = patch
	}

	// Track content to add to section
	if updateBlock.AddVariable != nil && updateBlock.AddVariable.Content != "" {
		u.addToSection[updateBlock.AddVariable.Section] += updateBlock.AddVariable.Content
	}

	// Mark update as processed
	delete(u.updateMap, u.varState.Key)
	u.Logger.Debug("applied variable update", "key", u.varState.Key, "patches", len(updateBlock.Patches))

	u.varState = nil

	return nil
}
