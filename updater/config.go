package updater

import (
	"log/slog"
	"maps"
)

type UpdateMode uint8

const (
	ModeReplace UpdateMode = 1 << iota
	ModeAdd
	ModeMoveSection
)

type Config struct {
	Logger        *slog.Logger
	Mode          UpdateMode
	EnsureNewLine bool
	DefaultQuote  byte

	SectionStartComments map[string]string
	SectionEndComments   map[string]string
}

var DefaultConfig = &Config{
	Logger:               slog.Default(),
	Mode:                 ModeReplace | ModeAdd | ModeMoveSection,
	EnsureNewLine:        true,
	DefaultQuote:         '"',
	SectionStartComments: make(map[string]string),
	SectionEndComments:   make(map[string]string),
}

type Option func(*Config)

func SetLogger(l *slog.Logger) Option {
	return func(c *Config) { c.Logger = l }
}

func SetEnsureNewLine(v bool) Option {
	return func(c *Config) { c.EnsureNewLine = v }
}

// As a parameter takes map of section name : comment
// If section name is empty -- applied by default for every section.
func SetSectionStartComments(comments map[string]string) Option {
	return func(c *Config) { maps.Copy(c.SectionStartComments, comments) }
}

// As a parameter takes map of section name : comment
// If section name is empty -- applied by default for every section.
func SetSectionEndComments(comments map[string]string) Option {
	return func(c *Config) { maps.Copy(c.SectionEndComments, comments) }
}

func SetReplace(v bool) Option {
	return func(c *Config) { setFlag(&c.Mode, ModeReplace, v) }
}

func SetAdd(v bool) Option {
	return func(c *Config) { setFlag(&c.Mode, ModeAdd, v) }
}

func SetMoveSection(v bool) Option {
	return func(c *Config) { setFlag(&c.Mode, ModeMoveSection, v) }
}

func setFlag(m *UpdateMode, mask UpdateMode, on bool) {
	if on {
		*m |= mask
	} else {
		*m &^= mask
	}
}
