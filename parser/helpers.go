package parser

import (
	"bufio"

	"github.com/4nd3r5on/go-envfile/common"
)

func Parse(p Parser, s *bufio.Scanner) ([]common.ParsedLine, error) {
	lines := make([]common.ParsedLine, 0)

	for s.Scan() {
		line, err := p.ParseLine(s.Text())
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}

	return lines, nil
}
