package botx

import "golang.org/x/exp/slog"

// Options defines options for Bot.
type Options struct {
	Workers int
	Logger  *slog.Logger
}

// Option defines a function that configures Bot.
type Option func(*Options)

// WithWorkers sets the number of workers to run.
func WithWorkers(workers int) Option {
	return func(o *Options) { o.Workers = workers }
}

// WithLogger sets the logger to use.
func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) { o.Logger = logger }
}
