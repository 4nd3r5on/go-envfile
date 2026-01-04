package updater

import (
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/4nd3r5on/go-envfile/common"
)

// FromStream processes a parser stream and generates patches based on the provided updates.
// It returns a map of patches keyed by line index to be applied to the original content.
func FromStream(
	s common.ParserStream,
	updates []Update,
	options ...Option,
) (map[int64]common.Patch, error) {
	var lineIdx int64

	updater, err := NewUpdater(updates, options...)
	if err != nil {
		return nil, err
	}

	for {
		lineIdx = s.GetLineIdx()
		parsedLine, err := s.Next()

		if errors.Is(err, io.EOF) {
			return updater.HandleEOF(lineIdx)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to parse line %d: %w", lineIdx, err)
		}

		if err = updater.HandleParsedLine(lineIdx, parsedLine); err != nil {
			return nil, err
		}
	}
}

// parseUntilTerminated reads continuation lines until the value is terminated.
func parseUntilTerminated(s common.ParserStream, startLine int64, varKey string, logger *slog.Logger) ([]common.ParsedLine, error) {
	buf := make([]common.ParsedLine, 0)

	for {
		parsedLine, err := s.Next()
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("unexpected end of input: multi-line value for %q starting at line %d was not terminated", varKey, startLine)
		}

		if err != nil {
			return nil, fmt.Errorf("error reading continuation line: %w", err)
		}

		if parsedLine.VariableValPart == nil {
			return nil, fmt.Errorf("continuation line missing value part (line type: %d)", parsedLine.Type)
		}

		buf = append(buf, parsedLine)

		if parsedLine.VariableValPart.IsTerminated {
			logger.Debug("multi-line value terminated", "key", varKey, "total_continuation_lines", len(buf))

			return buf, nil
		}
	}
}
