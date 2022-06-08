package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/oklog/run"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/peterbourgon/ff/v3/fftoml"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type RootConfig struct {
	ConfigFile string
	Debug      bool
	RootDir    string
	RootUser   string

	help bool
}

var homedir string

func init() {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	homedir = u.HomeDir
}

// flags
var (
	logger      *zap.Logger
	rootFlagSet = flag.NewFlagSet("project", flag.ExitOnError)
)

func parseRootConfig(args []string) (*RootConfig, error) {
	var cfg RootConfig

	defaultRootConfig := filepath.Join(homedir, ".projectrc")
	defaultRootProject := filepath.Join(homedir, "code")

	rootFlagSet.StringVar(&cfg.RootDir, "root", defaultRootProject, "root directory project")
	rootFlagSet.StringVar(&cfg.RootUser, "user", "", "root user project")
	rootFlagSet.StringVar(&cfg.ConfigFile, "config", defaultRootConfig, "root config project")
	rootFlagSet.BoolVar(&cfg.Debug, "debug", false, "increase log verbosity")

	err := ff.Parse(rootFlagSet, args,
		ff.WithEnvVarPrefix("PROJECT"),
		ff.WithConfigFileFlag("config"),
		ff.WithAllowMissingConfigFile(true),
		ff.WithConfigFileParser(fftoml.Parser),
	)

	// expand path
	cfg.RootDir = expandPath(cfg.RootDir)
	cfg.ConfigFile = expandPath(cfg.ConfigFile)

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
	logger = initLogger(rcfg.Debug)
	defer logger.Sync()

	root := &ffcli.Command{
		Name:    "project [flags] <subcommand>",
		FlagSet: rootFlagSet,
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
		Subcommands: []*ffcli.Command{
			initCommand(rcfg),
			listCommand(rcfg),
			newCommand(rcfg),
			getCommand(rcfg),
			queryCommand(rcfg),
		},
	}

	// create process context
	processCtx, processCancel := context.WithCancel(context.Background())
	var process run.Group
	{
		// add root command to process
		process.Add(func() error {
			return root.ParseAndRun(processCtx, args)
		}, func(error) {
			processCancel()
		})
	}

	if strings.HasPrefix(rcfg.RootDir, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			logger.Fatal("unable to get home directory", zap.Error(err))
		}

		rcfg.RootDir = strings.Replace(rcfg.RootDir, "~", home, 1)
	}

	if _, err := os.Stat(rcfg.RootDir); os.IsNotExist(err) {
		fmt.Printf("creating %s\n", rcfg.RootDir)
		if err = os.MkdirAll(rcfg.RootDir, os.ModePerm); err != nil {
			logger.Fatal("mkdir error", zap.Error(err))
		}
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

func expandPath(path string) string {
	path = os.ExpandEnv(path)
	if strings.HasPrefix(path, "~") {
		return strings.Replace(path, "~", homedir, 1)
	}
	return path
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
