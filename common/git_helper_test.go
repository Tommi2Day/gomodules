package common

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/tommi2day/gomodules/test"

	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestGit(t *testing.T) {
	test.InitTestDirs()

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
	t.Run("TestIsGitFile", func(t *testing.T) {
		type testTableType struct {
			name     string
			file     string
			expected bool
			gitFile  string
		}
		for _, testconfig := range []testTableType{
			{
				name:     "full path",
				file:     path.Join(test.TestDir, "testinit.go"),
				expected: true,
				gitFile:  "test/testinit.go",
			},
			{
				name:     "short path with dir",
				file:     path.Join("test", "testinit.go"),
				expected: true,
				gitFile:  "test/testinit.go",
			},
			{
				name:     "root file",
				file:     "CHANGELOG.md",
				expected: true,
				gitFile:  "CHANGELOG.md",
			},
			{
				name:     "short path fail",
				file:     path.Join("test", "dummy.txt"),
				expected: false,
			},
		} {
			t.Run(testconfig.name, func(t *testing.T) {
				root, fn, err := IsGitFile(testconfig.file)
				if testconfig.expected {
					assert.NoErrorf(t, err, "IsGitFile failed for %s: %s", testconfig.file, err)
					assert.NotEmptyf(t, root, "Git root empty")
					assert.NotEmptyf(t, fn, "filename empty")
				} else {
					assert.Error(t, err, "Expected error not set")
				}
			})
		}
	})

	t.Run("TestGetLastCommit OK", func(t *testing.T) {
		if os.Getenv("SKIP_COMMIT") == "true" {
			t.Skip("Skipping on CI")
		}
		filename := path.Join(test.TestDir, "testinit.go")
		gitDir, gitName, err := IsGitFile(filename)
		t.Logf("GitRootDir: %s, filename: %s", gitDir, gitName)
		assert.NoErrorf(t, err, "IsGitFile failed: %s", err)
		c, err := GetLastCommit(gitDir, gitName)
		assert.NoErrorf(t, err, "GetLastCommit failed: %s", err)
		require.IsTypef(t, &object.Commit{}, c, "GetLastCommit returned wrong type")
		if c == nil {
			t.Fatal("GetLastCommit returned nil")
		} else {
			hs := c.Hash.String()
			assert.NotEmptyf(t, hs, "Hash empty")
			m := c.Message
			assert.NotEmptyf(t, m, "Message empty")
			author := c.Author.Name
			assert.NotEmptyf(t, author, "Author empty")
			ct := c.Committer.When
			assert.Greaterf(t, ct.Unix(), int64(0), "Commit time empty")
			cts := ct.Format("02.01.2006 15:04")
			commit := fmt.Sprintf("Commit: %s has been committed by %s at %s (%s) with message '%s'", gitName, author, cts, hs[0:8], strings.TrimSuffix(m, "\n"))
			t.Log(commit)
		}
	})
	t.Run("TestNonGit ERROR", func(t *testing.T) {
		filename := path.Join(test.TestData, "testgit.txt")
		err := WriteStringToFile(filename, "test")
		require.NoErrorf(t, err, "WriteFile failed")
		gitDir, err := GetGitRootDir(filename)
		assert.NoErrorf(t, err, "GetGitRootDir should not fail: %s", err)
		t.Logf("GitRootDir: %s, filename: %s", gitDir, filename)
		c, err := GetLastCommit(gitDir, filename)
		assert.Errorf(t, err, "GetLastCommit should fail")
		assert.Nilf(t, c, "GetLastCommit returned not nil")
	})
	t.Run("TestGetJobURL", func(t *testing.T) {
		t.Run("With CI Job URL", func(t *testing.T) {
			expectedURL := "https://ci.example.com/jobs/123"
			_ = os.Setenv("CI_JOB_URL", expectedURL)
			result := GetGitlabJobURL()
			assert.Equal(t, expectedURL, result)
		})

		t.Run("Without CI Job URL", func(t *testing.T) {
			_ = os.Unsetenv("CI_JOB_URL")
			result := GetGitlabJobURL()
			assert.Empty(t, result)
		})

		t.Run("With Empty CI Job URL", func(t *testing.T) {
			_ = os.Setenv("CI_JOB_URL", "")
			result := GetGitlabJobURL()
			assert.Empty(t, result)
		})

		t.Run("With Special Characters in URL", func(t *testing.T) {
			specialURL := "https://ci.example.com/jobs/123?branch=feature/test¶m=value#fragment"
			_ = os.Setenv("CI_JOB_URL", specialURL)
			result := GetGitlabJobURL()
			assert.Equal(t, specialURL, result)
		})
		_ = os.Unsetenv("CI_JOB_URL")
	})

	t.Run("TestGetPipelineURL", func(t *testing.T) {
		t.Run("With Pipeline URL", func(t *testing.T) {
			expected := "https://gitlab.example.com/pipeline/123"
			_ = os.Setenv("CI_PIPELINE_URL", expected)
			result := GetGitlabPipelineURL()
			assert.Equal(t, expected, result)
		})

		t.Run("Without Pipeline URL", func(t *testing.T) {
			_ = os.Unsetenv("CI_PIPELINE_URL")
			result := GetGitlabPipelineURL()
			assert.Empty(t, result)
		})

		t.Run("With Empty Pipeline URL", func(t *testing.T) {
			_ = os.Setenv("CI_PIPELINE_URL", "")
			result := GetGitlabPipelineURL()
			assert.Empty(t, result)
		})

		t.Run("With Special Characters", func(t *testing.T) {
			expected := "https://gitlab.example.com/pipeline/123?branch=feature/test&commit=abc123"
			_ = os.Setenv("CI_PIPELINE_URL", expected)
			result := GetGitlabPipelineURL()
			assert.Equal(t, expected, result)
		})
		_ = os.Unsetenv("CI_PIPELINE_URL")
	})
}
