package envfile

import (
	"bufio"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/4nd3r5on/go-envfile/common"
	"github.com/4nd3r5on/go-envfile/parser"
	"github.com/4nd3r5on/go-envfile/updater"
	"github.com/safeblock-dev/werr"
)

type UpdateFileOptions struct {
	Backup               bool
	Logger               *slog.Logger
	SectionStartComments map[string]string
	SectionEndComments   map[string]string
}

func UpdateFile(
	path string,
	updates []updater.Update,
	opts UpdateFileOptions,
) error {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.Backup {
		if err := common.CreateBackup(opts.Logger, path); err != nil {
			return werr.Wrapf(err, "error trying to create backup for file %q", path)
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return werr.Wrapf(err, "error trying to open file %q", path)
	}

	p := parser.NewFileParser(nil, bufio.NewReader(file), false, parser.SetLogger(opts.Logger))

	patches, err := updater.FromStream(p,
		updates,
		updater.SetLogger(opts.Logger),
		updater.SetSectionStartComments(opts.SectionStartComments),
		updater.SetSectionEndComments(opts.SectionStartComments),
	)
	if err != nil {
		return werr.Wrapf(err, "failed to create patches %q", path)
	}

	if err = file.Close(); err != nil {
		return werr.Wrapf(err, "failed to close file %q", path)
	}

	opts.Logger.Info("patches summary", "count", len(patches))
	for i, patch := range patches {
		if patch.ShouldInsert {
			opts.Logger.Debug(
				"patch insert before",
				"patch", i,
				"content", patch.Insert,
				"length", len(patch.Insert),
			)
		}
		if patch.ShouldInsertAfter {
			opts.Logger.Debug(
				"patch insert after",
				"patch", i,
				"content", patch.InsertAfter,
				"length", len(patch.InsertAfter),
			)
		}
	}

	err = common.ApplyPatches(path, patches, false, opts.Logger)
	return werr.Wrapf(err, "failed to apply patched %q", path)
}

// Alias for creating parser
func NewParser(opts ...parser.Option) *parser.Parser {
	return parser.New(opts...)
}

// Parse everything from a scanner to an array or parsed lines
func Parse(p common.Parser, s *bufio.Scanner) ([]common.ParsedLine, error) {
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

func ParseFile(filePath string, p common.Parser) ([]common.ParsedLine, error) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	return Parse(p, bufio.NewScanner(file))
}

// LinesToVariableMap converts an array of ParsedLine into a map of variable key-value pairs.
// It handles multiline variables by accumulating unterminated values across lines.
func LinesToVariableMap(lines []common.ParsedLine) map[string]string {
	result := make(map[string]string)

	var currentKey string
	var currentValue strings.Builder
	var inMultiline bool

	for _, line := range lines {
		switch line.Type {
		case common.LineTypeVar:
			// If we were building a multiline variable, finalize it
			if inMultiline && currentKey != "" {
				result[currentKey] = currentValue.String()
				currentValue.Reset()
				inMultiline = false
			}

			// Start new variable
			if line.Variable != nil {
				currentKey = line.Variable.Key
				currentValue.WriteString(line.Variable.Value)

				if line.Variable.IsTerminated {
					// Single-line variable, store immediately
					result[currentKey] = currentValue.String()
					currentValue.Reset()
					currentKey = ""
				} else {
					// Multiline variable starts
					inMultiline = true
				}
			}

		case common.LineTypeVal:
			// Continuation of a multiline variable
			if inMultiline && line.VariableValPart != nil {
				// Add newline before appending next part (preserve multiline format)
				currentValue.WriteString("\n")
				currentValue.WriteString(line.VariableValPart.Value)

				if line.VariableValPart.IsTerminated {
					// Multiline variable ends
					result[currentKey] = currentValue.String()
					currentValue.Reset()
					currentKey = ""
					inMultiline = false
				}
			}
		}
	}

	// Handle case where file ends with unterminated variable
	if inMultiline && currentKey != "" {
		result[currentKey] = currentValue.String()
	}

	return result
}
