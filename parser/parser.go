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
// Lines from the file must be passed sequentially.
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

// handleUnterminatedValue processes continuation lines for unterminated multi-line values.
func (p *Parser) handleUnterminatedValue(line string) (common.ParsedLine, error) {
	terminator := FindTerminator(line, 0, p.terminator)

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
				Quote:        p.terminator,
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
			Quote:        p.terminator,
		},
		UnterminatedValueLines: p.unterminatedValueLines + 1,
	}

	// Reset state after finding terminator
	p.unterminatedValueLines = 0
	p.terminator = 0

	return parsedLine, nil
}

// handleCommentLine processes comment lines, including section markers.
func (p *Parser) handleCommentLine(line string) (common.ParsedLine, error) {
	if !p.IgnoreSections {
		if isSectionStart, data := common.MatchSectionStart(line); isSectionStart {
			return p.handleSectionStart(line, data.Name, data.Comment), nil
		}

		if isSectionEnd, data := common.MatchSectionEnd(line); isSectionEnd {
			return p.handleSectionEnd(line, data.Name, data.Comment), nil
		}
	}

	return common.ParsedLine{
		Type:        common.LineTypeComment,
		RawLine:     line,
		SectionData: p.currentSection,
	}, nil
}

// handleSectionStart processes section start markers.
func (p *Parser) handleSectionStart(line, name, comment string) common.ParsedLine {
	p.currentSection = &common.SectionData{
		Variables: make(map[string]struct{}),
		Name:      name,
	}

	return common.ParsedLine{
		Type:                         common.LineTypeSectionStart,
		RawLine:                      line,
		SectionData:                  p.currentSection,
		SectionStartEndInlineComment: comment,
	}
}

// handleSectionEnd processes section end markers.
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

// handleRawLine processes raw (non-parsed) lines.
func (p *Parser) handleRawLine(line string) common.ParsedLine {
	return common.ParsedLine{
		Type:        common.LineTypeRaw,
		RawLine:     line,
		SectionData: p.currentSection,
	}
}

// handleVariableLine processes variable assignment lines.
func (p *Parser) handleVariableLine(line string) (common.ParsedLine, error) {
	data, err := ParseVariable(line)
	if err != nil {
		return common.ParsedLine{}, err
	}

	if p.currentSection != nil {
		p.currentSection.Variables[data.Key.Key] = struct{}{}
	}

	isQuoted, terminator := GetQuoteFromValType(data.Value.Type)

	if !data.Value.IsTerminated {
		p.unterminatedValueLines++
		p.terminator = terminator
	}

	var suffix string

	suffixStart := data.Value.End + 1
	if suffixStart < len(line) {
		suffix = line[suffixStart:]
	}

	return common.ParsedLine{
		Type:    common.LineTypeVar,
		RawLine: line,
		Variable: &common.VariableData{
			Key:          data.Key.Key,
			Value:        data.Value.Content,
			Prefix:       line[:data.Value.Start],
			Suffix:       suffix,
			IsTerminated: data.Value.IsTerminated,
			IsQuoted:     isQuoted,
			Quote:        terminator,
		},
		UnterminatedValueLines: p.unterminatedValueLines,
		SectionData:            p.currentSection,
	}, nil
}
