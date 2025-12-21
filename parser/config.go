package parser

import "log/slog"

type Config struct {
	Logger         *slog.Logger
	IgnoreSections bool
}

type Option func(*Config)

func SetLogger(l *slog.Logger) Option {
	return func(c *Config) {
		c.Logger = l
	}
}

func SetIgnoreSections(ignore bool) Option {
	return func(c *Config) {
		c.IgnoreSections = ignore
	}
}
