package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/oklog/run"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type RootConfig struct {
	Verbose    bool
	RootDir    string
	RootUser   string
	ConfigFile string
}

// flags
var (
	logger      *zap.Logger
	rootFlagSet = flag.NewFlagSet("projects", flag.ExitOnError)
)

func parseRootConfig(args []string) (*RootConfig, error) {
	var cfg RootConfig
	rootFlagSet.StringVar(&cfg.RootDir, "root", "~/code", "root directory project")
	rootFlagSet.StringVar(&cfg.RootUser, "user", "", "root user project")
	rootFlagSet.StringVar(&cfg.ConfigFile, "config", "", "root config project")
	rootFlagSet.BoolVar(&cfg.Verbose, "v", false, "increase log verbosity")
	err := ff.Parse(rootFlagSet, args, ff.WithConfigFileFlag("config"))
	if err != nil {
		return nil, fmt.Errorf("unable to parse flags: %w", err)
	}

	return &cfg, nil
}

func main() {
	args := os.Args[1:]

	rcfg, err := parseRootConfig(args)
	if err != nil {
		panic(err)
	}

	// init logger
	logger = initLogger(rcfg.Verbose)
	defer logger.Sync()

	root := &ffcli.Command{
		Name:    "projects [flags] <subcommand>",
		FlagSet: rootFlagSet,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			return nil
		},
		Subcommands: []*ffcli.Command{
			listCommand(rcfg),
			newCommand(rcfg),
		},
	}

	// create process context
	processCtx, processCancel := context.WithCancel(context.Background())
	var process run.Group
	{
		// handle interrupt signals
		execute, interrupt := run.SignalHandler(processCtx, os.Interrupt)
		process.Add(execute, interrupt)

		// add root command to process
		process.Add(func() error {
			return root.ParseAndRun(processCtx, args)
		}, func(error) {
			processCancel()
		})
	}

	// start process
	switch err := process.Run(); err {
	case flag.ErrHelp, nil: // ok
	case context.Canceled, context.DeadlineExceeded:
		logger.Error("interrupted", zap.Error(err))
	default:
		logger.Fatal(err.Error())
	}
}

func initLogger(verbose bool) *zap.Logger {
	var level zapcore.Level
	if verbose {
		level = zapcore.DebugLevel
	} else {
		level = zapcore.InfoLevel
	}

	encodeConfig := zap.NewDevelopmentEncoderConfig()
	encodeConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encodeConfig.EncodeTime = nil
	consoleEncoder := zapcore.NewConsoleEncoder(encodeConfig)
	consoleDebugging := zapcore.Lock(os.Stdout)
	core := zapcore.NewCore(consoleEncoder, consoleDebugging, level)
	logger := zap.New(core)

	logger.Debug("logger initialised")
	return logger
}
