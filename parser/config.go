package parser

import "log/slog"

type Config struct {
	Logger           *slog.Logger
	IgnoreSections   bool
	PreserveComments bool
}

type Option func(*Config)

func WithLogger(l *slog.Logger) Option {
	return func(c *Config) {
		c.Logger = l
	}
}
