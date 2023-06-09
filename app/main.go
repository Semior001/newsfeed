// Package main is an entrypoint for application
package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/Semior001/newsfeed/app/cmd"
	"github.com/Semior001/newsfeed/pkg/logx"
	"github.com/jessevdk/go-flags"
	"golang.org/x/exp/slog"
)

var opts struct {
	Run      cmd.Run `command:"run" description:"run newsfeed bot"`
	JSONLogs bool    `long:"json-logs" env:"JSON_LOGS" description:"turn on json logs"`
	Debug    bool    `long:"dbg" env:"DEBUG" description:"turn on debug mode"`
}

var version = "unknown"

func getVersion() string {
	v, ok := debug.ReadBuildInfo()
	if !ok || v.Main.Version == "(devel)" {
		return version
	}
	return v.Main.Version
}

func main() {
	fmt.Printf("newsfeed, version: %s\n", getVersion())

	p := flags.NewParser(&opts, flags.Default)
	p.CommandHandler = func(cmd flags.Commander, args []string) error {
		setupLog()

		if err := cmd.Execute(args); err != nil {
			slog.Error("failed to execute command", slog.Any("err", err))
			os.Exit(1)
		}

		return nil
	}

	// after failure command does not return non-zero code
	if _, err := p.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			slog.Error("failed to parse flags", slog.Any("err", err))
			os.Exit(1)
		}
	}
}

func setupLog() {
	handler := slog.HandlerOptions{
		AddSource:   false,
		Level:       slog.LevelInfo,
		ReplaceAttr: nil,
	}

	if opts.Debug {
		handler.Level = slog.LevelDebug
		handler.AddSource = true
	}

	h := slog.Handler(handler.NewTextHandler(os.Stderr))

	if opts.JSONLogs {
		h = handler.NewJSONHandler(os.Stderr)
	}

	lg := slog.New(&logx.Chain{
		Handler: h,
		Middleware: []logx.Middleware{
			logx.RequestID(),
			logx.StacktraceOnError(),
		},
	})
	slog.SetDefault(lg)
}
