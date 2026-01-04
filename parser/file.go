package parser

import (
	"bufio"
	"bytes"

	"github.com/4nd3r5on/go-envfile/common"
)

type FileParser struct {
	common.Parser

	reader      *bufio.Reader
	CurrentIdx  int64
	keepNewLine bool
}

func NewFileParser(p common.Parser, reader *bufio.Reader, keepNewLine bool, options ...Option) *FileParser {
	if p == nil {
		p = New(options...)
	}

	return &FileParser{
		Parser:      p,
		reader:      reader,
		CurrentIdx:  0,
		keepNewLine: keepNewLine,
	}
}

func (p *FileParser) Next() (common.ParsedLine, error) {
	line, err := common.ReadLineWithEOL(p.reader)
	if err != nil {
		return common.ParsedLine{}, err
	}

	p.CurrentIdx++

	if p.keepNewLine {
		return p.ParseLine(string(line))
	}

	clean := bytes.TrimRight(line, "\r\n")

	return p.ParseLine(string(clean))
}

func (p *FileParser) GetLineIdx() int64 { return p.CurrentIdx }
