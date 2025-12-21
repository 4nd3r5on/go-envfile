package envfile

import (
	"bufio"
	"fmt"
	"log"
	"log/slog"
	"os"

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
	updates []common.Update,
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
		updater.SetSectionEndComments(opts.SectionStartComments))
	if err != nil {
		return werr.Wrapf(err, "failed to create patches %q", path)
	}

	if err = file.Close(); err != nil {
		return werr.Wrapf(err, "failed to close file %q", path)
	}

	log.Println("Patches count:", len(patches))
	for i, patch := range patches {
		if patch.ShouldInsert {
			fmt.Printf("P%d insert before\n%s", i+1, patch.Insert)
		}
		if patch.ShouldInsertAfter {
			fmt.Printf("P%d insert after\n%s", i+1, patch.InsertAfter)
		}
	}

	err = common.ApplyPatches(path, patches, false, opts.Logger)
	return werr.Wrapf(err, "failed to apply patched %q", path)
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
