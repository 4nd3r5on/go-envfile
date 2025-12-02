package updater

import (
	"fmt"
	"log/slog"

	"github.com/4nd3r5on/go-envfile/common"
)

var DefaultConfig = &Config{
	Logger: slog.Default(),
}

type Updater struct {
	*Config
	updates map[string]common.Update
	patches map[int64]common.Patch
}

func New(options ...Option) *Updater {
	u := &Updater{
		Config: DefaultConfig,
	}

	for _, option := range options {
		option(u.Config)
	}
	return u
}

func (u *Updater) AddUpdates(updates []common.Update) error {
	for _, update := range updates {
		if _, seen := u.updates[update.Key]; seen {
			return fmt.Errorf("")
		}
		u.updates[update.Key] = update
	}
	return nil
}

func (u *Updater) FromStream(s common.ParserStream) (map[int64]common.Patch, error) {
	var lineIdx int64 = 0
	return u.patches, nil
}
