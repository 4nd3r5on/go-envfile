package parser

import (
	"log/slog"

	"github.com/4nd3r5on/go-envfile/common"
)

var DefaultConfig = &Config{
	Logger:           slog.Default(),
	IgnoreSections:   false,
	PreserveComments: true,
}

type Parser struct {
	*Config
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

func (p *Parser) ParseLine(line string) (common.ParsedLine, error) {
	return common.ParsedLine{}, nil
}

func (p *Parser) FromStream() {
}
