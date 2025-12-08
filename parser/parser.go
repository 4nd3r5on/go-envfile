package parser

import (
	"errors"
	"log/slog"

	"github.com/4nd3r5on/go-envfile/common"
)

var DefaultConfig = &Config{
	Logger:         slog.Default(),
	IgnoreSections: false,
}

type Parser struct {
	*Config
	currentSection         *common.SectionData
	unterminatedValueLines int
	terminator             byte
}

func New(options ...Option) *Parser {
	p := &Parser{Config: DefaultConfig}

	for _, option := range options {
		option(p.Config)
	}
	return p
}

// ParseLine takes as an input line from an environment file and outputs parsed line
// Lines from the file must be passed sequentially
func (p *Parser) ParseLine(line string) (common.ParsedLine, error) {
	if p.unterminatedValueLines > 0 {
		return p.handleUnterminatedValue(line)
	}
	lineType := DetectLineType(line)
	switch lineType {
	case common.LineTypeComment:
		return p.handleCommentLine(line)
	case common.LineTypeRaw:
		return p.handleRawLine(line), nil
	case common.LineTypeVar:
		return p.handleVariableLine(line)
	default:
		return common.ParsedLine{}, errors.New("unexpected line type")
	}
}

// handleUnterminatedValue processes continuation lines for unterminated multi-line values
func (p *Parser) handleUnterminatedValue(line string) (common.ParsedLine, error) {
	terminator := findTerminator(line, 0, p.terminator)

	if terminator < 0 {
		// Value continues on next line
		p.unterminatedValueLines++
		return common.ParsedLine{
			Type:    common.LineTypeVal,
			RawLine: line,
			VariableValPart: &common.VariableValPartData{
				Value:        line,
				Suffix:       "",
				IsTerminated: false,
			},
			UnterminatedValueLines: p.unterminatedValueLines,
		}, nil
	}

	// Value terminates on this line
	val := line[:terminator]
	suffix := ""
	if terminator+1 < len(line) {
		suffix = line[terminator+1:]
	}

	parsedLine := common.ParsedLine{
		Type:    common.LineTypeVal,
		RawLine: line,
		VariableValPart: &common.VariableValPartData{
			Value:        val,
			Suffix:       suffix,
			IsTerminated: true,
		},
		UnterminatedValueLines: p.unterminatedValueLines + 1,
	}

	// Reset state after finding terminator
	p.unterminatedValueLines = 0
	p.terminator = 0

	return parsedLine, nil
}

// handleCommentLine processes comment lines, including section markers
func (p *Parser) handleCommentLine(line string) (common.ParsedLine, error) {
	if !p.Config.IgnoreSections {
		if isSectionStart, name, comment := MatchSectionStart(line); isSectionStart {
			return p.handleSectionStart(line, name, comment), nil
		}
		if isSectionEnd, name, comment := MatchSectionEnd(line); isSectionEnd {
			return p.handleSectionEnd(line, name, comment), nil
		}
	}

	return common.ParsedLine{
		Type:        common.LineTypeComment,
		RawLine:     line,
		SectionData: p.currentSection,
	}, nil
}

// handleSectionStart processes section start markers
func (p *Parser) handleSectionStart(line, name, comment string) common.ParsedLine {
	p.currentSection = &common.SectionData{
		Name: name,
	}
	return common.ParsedLine{
		Type:                         common.LineTypeSectionStart,
		RawLine:                      line,
		SectionData:                  p.currentSection,
		SectionStartEndInlineComment: comment,
	}
}

// handleSectionEnd processes section end markers
func (p *Parser) handleSectionEnd(line, name, comment string) common.ParsedLine {
	parsedLine := common.ParsedLine{
		Type:                         common.LineTypeSectionEnd,
		RawLine:                      line,
		SectionData:                  p.currentSection,
		SectionStartEndInlineComment: comment,
	}

	if p.currentSection != nil && name == p.currentSection.Name {
		p.currentSection = nil
	}

	return parsedLine
}

// handleRawLine processes raw (non-parsed) lines
func (p *Parser) handleRawLine(line string) common.ParsedLine {
	return common.ParsedLine{
		Type:        common.LineTypeRaw,
		RawLine:     line,
		SectionData: p.currentSection,
	}
}

// handleVariableLine processes variable assignment lines
func (p *Parser) handleVariableLine(line string) (common.ParsedLine, error) {
	key, val, isTerminated, terminator, err := ParseVariable(line)
	if err != nil {
		return common.ParsedLine{}, err
	}

	if !isTerminated {
		p.unterminatedValueLines++
		p.terminator = terminator
	}

	return common.ParsedLine{
		Type:    common.LineTypeVar,
		RawLine: line,
		Variable: &common.VariableData{
			Key:          key.Val,
			Value:        val.Val,
			Prefix:       line[:val.Start],
			Suffix:       line[val.Start:val.End],
			IsTerminated: isTerminated,
		},
		UnterminatedValueLines: p.unterminatedValueLines,
		SectionData:            p.currentSection,
	}, nil
}
