package common

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/tommi2day/gomodules/test"

	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestGit(t *testing.T) {
	test.Testinit(t)

	t.Run("TestGetGitRootDir OK", func(t *testing.T) {
		dir, err := GetGitRootDir(test.TestDir)
		t.Logf("GitRootDir: %s", dir)
		assert.NoErrorf(t, err, "GetGitRootDir failed: %s", err)
		assert.NotEmptyf(t, dir, "dir empty")
	})
	t.Run("TestGetGitRootDir Error", func(t *testing.T) {
		dir, err := GetGitRootDir("/tmp")
		t.Logf("GitRootDir: %s", dir)
		assert.Errorf(t, err, "GetGitRootDir should fail")
	})
	t.Run("TestGetLastCommit OK", func(t *testing.T) {
		_, filename, _, _ := runtime.Caller(0)
		// filename := path.Join(test.TestDir, "testinit.go")
		gitDir, err := GetGitRootDir(filename)
		require.NoErrorf(t, err, "GetGitRootDir failed: %s", err)
		t.Logf("Testfile: %s", filename)
		err = IsGitFile(filename)
		assert.NoErrorf(t, err, "IsGitFile failed: %s", err)
		c, err := GetLastCommit(gitDir, filename)
		assert.NoErrorf(t, err, "GetLastCommit failed: %s", err)
		require.IsTypef(t, &object.Commit{}, c, "GetLastCommit returned wrong type")
		if c == nil {
			t.Fatal("GetLastCommit returned nil")
		}
		hs := c.Hash.String()
		assert.NotEmptyf(t, hs, "Hash empty")
		m := c.Message
		assert.NotEmptyf(t, m, "Message empty")
		ct := c.Committer.When
		assert.Greaterf(t, ct.Unix(), int64(0), "Commit time empty")
		cts := ct.Format("02.01.2006 15:04")
		commit := fmt.Sprintf("Commit: %s has been committed at %s (%s) with message '%s'", strings.TrimPrefix(filename, gitDir+"/"), cts, hs[0:8], strings.TrimSuffix(m, "\n"))
		t.Log(commit)
	})
	t.Run("TestNonGit ERROR", func(t *testing.T) {
		filename := path.Join(test.TestData, "testgit.txt")
		err := os.WriteFile(filename, []byte("test"), 0600)
		require.NoErrorf(t, err, "WriteFile failed")
		gitDir, err := GetGitRootDir(filename)
		assert.NoErrorf(t, err, "GetGitRootDir should not fail: %s", err)
		t.Logf("GitRootDir: %s, filename: %s", gitDir, filename)
		c, err := GetLastCommit(gitDir, filename)
		assert.Errorf(t, err, "GetLastCommit should fail")
		assert.Nilf(t, c, "GetLastCommit returned not nil")
	})
}
