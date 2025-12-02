package envfile

import (
	"log/slog"

	"github.com/4nd3r5on/go-envfile/common"
	"github.com/4nd3r5on/go-envfile/parser"
	"github.com/4nd3r5on/go-envfile/updater"
)

func UpdateFile(
	path string,
	updates []common.Update,
	backup bool,
	logger *slog.Logger,
) error {
	if logger == nil {
		logger = slog.Default()
	}
	if backup {
		if err := common.CreateBackup(logger, path); err != nil {
			return err
		}
	}

	p, err := parser.NewFileParser(nil, path, parser.WithLogger(logger))
	if err != nil {
		return err
	}

	patches, err := updater.New(
		updater.WithLogger(logger),
		updater.WithReplace(true),
		updater.WithAdd(true),
		updater.WithMoveSection(true),
	).FromStream(p)
	if err != nil {
		return err
	}
	return common.ApplyPatches(path, patches)
}
