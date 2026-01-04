package updater

import (
	"errors"
	"fmt"
	"io"

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
