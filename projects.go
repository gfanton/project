package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

type Project struct {
	Path         string
	Name         string
	Organisation string
}

func (p *Project) git() string {
	return filepath.Join(p.Path, ".git")
}

func (p *Project) IsGit() bool {
	_, err := os.Stat(p.git())
	return !os.IsNotExist(err)
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
