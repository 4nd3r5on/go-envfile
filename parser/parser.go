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
	p := &Parser{
		Config: DefaultConfig,
	}

	for _, option := range options {
		option(p.Config)
	}
	return p
}

// ParseLine takes as an input line from an environment file and outputs parsed line
// Lines from the file must be passed sequentially
func (p *Parser) ParseLine(line string) (common.ParsedLine, error) {
	if p.unterminatedValueLines > 0 {
		terminator := findTerminator(line, 0, p.terminator)
		if terminator < 0 {
			p.unterminatedValueLines++
			return common.ParsedLine{}, nil
		} else {
			p.unterminatedValueLines = 0
			p.terminator = 0
			return common.ParsedLine{}, nil
		}
	}

	lineType := DetectLineType(line)
	switch lineType {
	case common.LineTypeComment:
		// TODO: Try to parse as a section
		return common.ParsedLine{
			Type:        common.LineTypeComment,
			RawLine:     line,
			SectionData: p.currentSection,
		}, nil
	case common.LineTypeRaw:
		return common.ParsedLine{
			Type:        lineType,
			RawLine:     line,
			SectionData: p.currentSection,
		}, nil
	case common.LineTypeVar:
		key, val, isTerminated, terminator, err := ParseVariable(line)
		if err != nil {
			return common.ParsedLine{}, err
		}
		if !isTerminated {
			p.unterminatedValueLines++
			p.terminator = terminator
		} else {
			p.unterminatedValueLines = 0
			p.terminator = 0
		}

		return common.ParsedLine{
			Type:    common.LineTypeVar,
			RawLine: line,
			Variable: &common.VariableData{
				Key:          key.Val,
				Value:        val.Val,
				Prefix:       line[:val.Start],
				Suffix:       line[:val.End],
				IsTerminated: isTerminated,
			},
			UnterminatedValueLines: p.unterminatedValueLines,
			SectionData:            p.currentSection,
		}, nil
	default:
		return common.ParsedLine{}, errors.New("unexpected line type")
	}
}
