package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

const GithubProvider = "github.com"

const DefaultProvider = GithubProvider

type Project struct {
	Path         string
	Name         string
	Organisation string
}

func ParseProject(rcfg *RootConfig, name string) (*Project, error) {
	split := strings.Split(name, string(os.PathSeparator))
	switch len(split) {
	case 1:
		if rcfg.RootUser != "" {
			project := split[0]
			pathslice := []string{rcfg.RootDir, rcfg.RootUser, split[0]}
			return &Project{
				Path:         strings.Join(pathslice, string(os.PathSeparator)),
				Name:         project,
				Organisation: rcfg.RootUser,
			}, nil
		}

		return nil, fmt.Errorf("no default user defined")
	case 2:
		user, project := split[0], split[1]
		pathslice := []string{rcfg.RootDir, user, project}
		return &Project{
			Path:         strings.Join(pathslice, string(os.PathSeparator)),
			Name:         project,
			Organisation: user,
		}, nil
	case 3:
	default:
		// provider
	}

	return nil, fmt.Errorf("malformed project name `%.30s`", name)
}

func (p *Project) GitHTTPUrl() string {
	return fmt.Sprintf("https://%s/%s/%s.git", GithubProvider, p.Organisation, p.Name)
}

func (p *Project) GitSSHPUrl() string {
	return fmt.Sprintf("git@%s:%s/%s", GithubProvider, p.Organisation, p.Name)
}

func (p *Project) git() string {
	return filepath.Join(p.Path, ".git")
}

func (p *Project) IsGit() bool {
	_, err := os.Stat(p.git())
	return !os.IsNotExist(err)
}

func (p *Project) String() string {
	return fmt.Sprintf("%s/%s", p.Organisation, p.Name)
}

func (p *Project) OpenRepo() (*git.Repository, error) {
	return git.PlainOpen(p.git())
}

const depth = 1

type WalkProjectFunc func(d fs.DirEntry, project *Project) error

func WalkProject(rootdir string, fn WalkProjectFunc) error {
	return filepath.WalkDir(rootdir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		relpath, err := filepath.Rel(rootdir, path)
		if err != nil {
			return err
		}

		sep := strings.Count(relpath, string(os.PathSeparator))
		if sep < depth {
			return nil
		}

		if sep > depth || strings.HasPrefix(filepath.Base(relpath), ".git") {
			return fs.SkipDir
		}

		split := strings.Split(relpath, string(os.PathSeparator))
		project := &Project{
			Path:         path,
			Name:         split[1],
			Organisation: split[0],
		}

		return fn(d, project)
	})
}
