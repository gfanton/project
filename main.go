package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/peterbourgon/ff/v3/fftoml"
	"go.uber.org/zap"
)

type RootConfig struct {
	ConfigFile string
	Debug      bool
	RootDir    string
	RootUser   string

	help bool
}

var homedir string

// flags
var (
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	args := os.Args[1:]

	logWriter := io.Discard

	rcfg, err := parseRootConfig(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to parse config: %s", err)
		os.Exit(1)
	}

	if rcfg.Debug {
		logWriter = os.Stderr
	}
	logger := log.New(logWriter, "D: ", log.Lshortfile)

	root := &ffcli.Command{
		Name:    "project [flags] <subcommand>",
		FlagSet: rootFlagSet,
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
		Subcommands: []*ffcli.Command{
			initCommand(logger, rcfg),
			listCommand(logger, rcfg),
			newCommand(logger, rcfg),
			getCommand(logger, rcfg),
			queryCommand(logger, rcfg),
		},
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

	if err := root.ParseAndRun(ctx, args); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		}
		os.Exit(1)
	}
}

func expandPath(path string) string {
	path = os.ExpandEnv(path)
	if strings.HasPrefix(path, "~") {
		return strings.Replace(path, "~", homedir, 1)
	}
	return path
}

func init() {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	homedir = u.HomeDir
}
