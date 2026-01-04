package updater

import (
	"fmt"

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

type VariableState struct {
	DefinitionLine int64
	Key            string
	IsTerminated   bool
	LinesBuf       []common.ParsedLine
}

type Updater struct {
	*Config

	// input
	updateMap map[string]Update
	// updater state
	currentSection      string
	sectionsLastVarLine map[string]int64  // for locating where to place a patch for a section
	addToSection        map[string]string // for something that we need to move into another section
	varState            *VariableState
	// output
	patchMap map[int64]common.Patch
}

func NewUpdater(updates []Update, options ...Option) (*Updater, error) {
	cfg := DefaultConfig
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

	cfg.Logger.Info("starting stream processing", "total_updates", len(updates))

	return &Updater{
		Config:              cfg,
		updateMap:           updateMap,
		sectionsLastVarLine: make(map[string]int64),
		addToSection:        make(map[string]string),
		patchMap:            make(map[int64]common.Patch),
	}, nil
}

func (u *Updater) HandleParsedLine(lineIdx int64, parsedLine common.ParsedLine) error {
	switch parsedLine.Type {
	case common.LineTypeSectionStart:
		return u.handleSectionStart(lineIdx, parsedLine)
	case common.LineTypeSectionEnd:
		return u.handleSectionEnd(lineIdx, parsedLine)
	case common.LineTypeVar:
		return u.handleVar(lineIdx, parsedLine)
	case common.LineTypeVal:
		return u.handleValPart(lineIdx, parsedLine)
	default:
		return nil
	}
}

func (u *Updater) handleSectionStart(lineIdx int64, parsedLine common.ParsedLine) error {
	if parsedLine.SectionData == nil {
		return fmt.Errorf("line %d: section start detected but SectionData is nil", lineIdx)
	}

	u.currentSection = parsedLine.SectionData.Name
	u.sectionsLastVarLine[u.currentSection] = lineIdx
	u.Logger.Debug("entered section", "section", u.currentSection, "line", lineIdx)

	return nil
}

func (u *Updater) handleSectionEnd(lineIdx int64, parsedLine common.ParsedLine) error {
	u.currentSection = ""
	u.Logger.Debug("section end", "section", u.currentSection, "line", lineIdx)

	return nil
}
