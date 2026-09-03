package maillib

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteAttachmentDedup verifies that saving attachments with a filename that
// already exists in the download directory does not overwrite the existing file,
// but instead appends "-yyyymmdd-1", "-yyyymmdd-2", ... before the extension, the
// same way a browser download does.
func TestWriteAttachmentDedup(t *testing.T) {
	dir := t.TempDir()
	fn := path.Join(dir, "invoice.pdf")
	date := time.Now().Format("20060102")

	writeAttachment(fn, strings.NewReader("first"), false)
	writeAttachment(fn, strings.NewReader("second"), false)
	writeAttachment(fn, strings.NewReader("third"), false)

	first, err := os.ReadFile(fn)
	require.NoErrorf(t, err, "first attachment file should exist")
	assert.Equal(t, "first", string(first))

	second, err := os.ReadFile(path.Join(dir, fmt.Sprintf("invoice-%s-1.pdf", date)))
	require.NoErrorf(t, err, "second attachment file should exist with -yyyymmdd-1 suffix")
	assert.Equal(t, "second", string(second))

	third, err := os.ReadFile(path.Join(dir, fmt.Sprintf("invoice-%s-2.pdf", date)))
	require.NoErrorf(t, err, "third attachment file should exist with -yyyymmdd-2 suffix")
	assert.Equal(t, "third", string(third))
}

// TestWriteAttachmentDedupNoExtension verifies dedup also works for filenames without an extension.
func TestWriteAttachmentDedupNoExtension(t *testing.T) {
	dir := t.TempDir()
	fn := path.Join(dir, "readme")
	date := time.Now().Format("20060102")

	writeAttachment(fn, strings.NewReader("first"), false)
	writeAttachment(fn, strings.NewReader("second"), false)

	first, err := os.ReadFile(fn)
	require.NoErrorf(t, err, "first attachment file should exist")
	assert.Equal(t, "first", string(first))

	second, err := os.ReadFile(path.Join(dir, fmt.Sprintf("readme-%s-1", date)))
	require.NoErrorf(t, err, "second attachment file should exist with -yyyymmdd-1 suffix")
	assert.Equal(t, "second", string(second))
}

// TestWriteAttachmentOverwrite verifies that with overwrite enabled, saving an
// attachment with a colliding filename replaces the existing file instead of
// creating a new one with a suffix.
func TestWriteAttachmentOverwrite(t *testing.T) {
	dir := t.TempDir()
	fn := path.Join(dir, "invoice.pdf")

	writeAttachment(fn, strings.NewReader("first"), true)
	writeAttachment(fn, strings.NewReader("second"), true)

	content, err := os.ReadFile(fn)
	require.NoErrorf(t, err, "attachment file should exist")
	assert.Equal(t, "second", string(content))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "only the latest attachment version should be kept")
}
