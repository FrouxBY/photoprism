package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRel(t *testing.T) {
	t.Run("same", func(t *testing.T) {
		assert.Equal(t, "", Rel("/some/path", "/some/path"))
	})
	t.Run("short", func(t *testing.T) {
		assert.Equal(t, "/some/", Rel("/some/", "/some/path"))
	})
	t.Run("empty", func(t *testing.T) {
		assert.Equal(t, "", Rel("", "/some/path"))
	})
	t.Run("/some/path", func(t *testing.T) {
		assert.Equal(t, "foo/bar.baz", Rel("/some/path/foo/bar.baz", "/some/path"))
	})
	t.Run("/some/path/", func(t *testing.T) {
		assert.Equal(t, "foo/bar.baz", Rel("/some/path/foo/bar.baz", "/some/path/"))
	})
	t.Run("/some/path/bar", func(t *testing.T) {
		assert.Equal(t, "/some/path/foo/bar.baz", Rel("/some/path/foo/bar.baz", "/some/path/bar"))
	})
	t.Run("empty dir", func(t *testing.T) {
		assert.Equal(t, "/some/path/foo/bar.baz", Rel("/some/path/foo/bar.baz", ""))
	})
}

func TestFileName(t *testing.T) {
	t.Run("Test copy 3.jpg", func(t *testing.T) {
		result := FileName("testdata/Test (4).jpg", ".photoprism", Abs("testdata"), ".xmp", true)

		assert.Equal(t, "testdata/.photoprism/Test.xmp", result)
	})

	t.Run("Test (3).jpg", func(t *testing.T) {
		result := FileName("testdata/Test (4).jpg", ".photoprism", Abs("testdata"), ".xmp", false)

		assert.Equal(t, "testdata/.photoprism/Test (4).xmp", result)
	})

	t.Run("FOO.XMP", func(t *testing.T) {
		result := FileName("testdata/FOO.XMP", ".photoprism/sub", Abs("testdata"), ".jpeg", true)

		assert.Equal(t, "testdata/.photoprism/sub/FOO.jpeg", result)
	})

	t.Run("Test copy 3.jpg", func(t *testing.T) {
		tempDir := filepath.Join(os.TempDir(), HiddenPath)

		// t.Logf("TEMP DIR, ABS NAME: %s, %s", tempDir, Abs("testdata/Test (4).jpg"))

		result := FileName(Abs("testdata/Test (4).jpg"), tempDir, Abs("testdata"), ".xmp", true)

		assert.Equal(t, tempDir+"/Test.xmp", result)
	})

	t.Run("empty dir", func(t *testing.T) {
		result := FileName("testdata/FOO.XMP", "", Abs("testdata"), ".jpeg", true)

		assert.Equal(t, "testdata/FOO.jpeg", result)
	})
}
