package parser

import (
	"bufio"
	"io"
	"os"

	"github.com/4nd3r5on/go-envfile/common"
)

type FileParser struct {
	*Parser
	*bufio.Scanner
	File       *os.File
	CurrentIdx int64
}

func NewFileParser(p *Parser, path string, options ...Option) (*FileParser, error) {
	if p == nil {
		p = New(options...)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)

	return &FileParser{
		Parser:     p,
		File:       file,
		Scanner:    scanner,
		CurrentIdx: 0,
	}, nil
}

func (p *FileParser) Next() (common.ParsedLine, error) {
	if p.Scanner.Scan() {
		p.CurrentIdx++
		return p.Parser.ParseLine(p.Scanner.Text())
	}
	return common.ParsedLine{}, io.EOF
}

func (p *FileParser) Close() error {
	return p.File.Close()
}
