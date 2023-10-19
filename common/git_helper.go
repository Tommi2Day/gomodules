package common

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	log "github.com/sirupsen/logrus"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GetLastCommit returns a string with last commit details for given file
func GetLastCommit(gitDir string, filename string) (c *object.Commit, err error) {
	r, err := git.PlainOpen(gitDir)
	if err != nil {
		err = fmt.Errorf("failed to open git repo at %s: %s", gitDir, err)
		return
	}
	filename = strings.TrimPrefix(filename, gitDir+"/")
	cIter, err := r.Log(&git.LogOptions{FileName: &filename})
	if err != nil {
		err = fmt.Errorf("failed to run git log for %s: %s", filename, err)
		return
	}
	c, err = cIter.Next()
	if err != nil {
		if err == io.EOF {
			err = fmt.Errorf("no commit for %s found", filename)
		} else {
			err = fmt.Errorf("failed to fetch commit for %s: %s", filename, err)
			return
		}
		return
	}
	if c == nil {
		err = fmt.Errorf("no commit selected")
	}
	return
}

// GetGitRootDir returns the root directory of a git repository
func GetGitRootDir(start string) (rootDir string, err error) {
	rel := ""
	rootDir = ""
	startDir := start
	if IsFile(start) {
		startDir = filepath.Dir(startDir)
	}
	startDir = filepath.ToSlash(startDir)
	basePath := filepath.VolumeName(startDir) + "/"
	targetPath := startDir
	for {
		rel, err = filepath.Rel(basePath, targetPath)
		if err != nil {
			err = fmt.Errorf("getGetRootDir RelPath Error: %s", err)
			return
		}
		// Exit the loop once we reach the basePath.
		if rel == "." {
			err = fmt.Errorf("GetGitRootDir: not found")
			return
		}
		rootDir, err = filepath.Abs(targetPath)
		if err != nil {
			err = fmt.Errorf("GetGetRootDir AbsPath Error: %s", err)
			return
		}
		rootDir = filepath.ToSlash(rootDir)
		if FileExists(filepath.Join(rootDir, ".git")) {
			return
		}
		// Going up!
		targetPath = filepath.Dir(targetPath)
	}
}

// IsGitFile returns nil if given file is part of a git repo or a meaningful message
func IsGitFile(filename string) (err error) {
	var tree *object.Tree
	var commit *object.Commit
	var ref *plumbing.Reference
	var fileList []string

	// fix path separator to be comparable
	filename = filepath.ToSlash(filename)

	// ... get git root dir
	rootDir := ""
	rootDir, err = GetGitRootDir(filename)
	if err != nil {
		err = fmt.Errorf("cannot find Git Root for %s: %s", rootDir, err)
		return
	}

	// ... get git filename
	gitName, _ := strings.CutPrefix(filename, rootDir+"/")
	repo, err := git.PlainOpen(rootDir)
	if err != nil {
		err = fmt.Errorf("cannot open git repo %s: %s", rootDir, err)
		return
	}

	// get repo head commit
	ref, err = repo.Head()
	if err == nil {
		// ... retrieving the commit object
		commit, err = repo.CommitObject(ref.Hash())
	}

	// ... retrieve the tree from the commit
	if err == nil {
		tree, err = commit.Tree()
	}

	// ... iterate over tree
	if err == nil {
		err = tree.Files().ForEach(func(f *object.File) error {
			// fmt.Println(f.Name)
			fileList = append(fileList, f.Name)
			return nil
		})
	}
	if err != nil {
		err = fmt.Errorf("error in tree.files: %s", err)
		return
	}
	// ... check if filename is in tree
	if have, _ := InArray(gitName, fileList); have {
		log.Debugf("IsGitFile: %s found in %s", gitName, rootDir)
		return nil
	}

	err = fmt.Errorf("%s not found in %s", gitName, rootDir)
	return
}
