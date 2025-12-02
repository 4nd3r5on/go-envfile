package updater

import "log/slog"

type UpdateMode uint8

const (
	ModeReplace UpdateMode = 1 << iota
	ModeAdd
	ModeMoveSection
)

type Config struct {
	Logger *slog.Logger
	Mode   UpdateMode
}

type Option func(*Config)

func WithLogger(l *slog.Logger) Option {
	return func(c *Config) {
		c.Logger = l
	}
}

func WithReplace(v bool) Option {
	return func(c *Config) {
		setFlag(&c.Mode, ModeReplace, v)
	}
}

func WithAdd(v bool) Option {
	return func(c *Config) {
		setFlag(&c.Mode, ModeAdd, v)
	}
}

func WithMoveSection(v bool) Option {
	return func(c *Config) {
		setFlag(&c.Mode, ModeMoveSection, v)
	}
}

func setFlag(m *UpdateMode, mask UpdateMode, on bool) {
	if on {
		*m |= mask
	} else {
		*m &^= mask
	}
}
