package localtest

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/AlekSi/shoulda/musta"
)

// Setup copies testdata cache to a test-specific temporary directory.
func Setup(t testing.TB) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	musta.BeTrue(t, ok)
	src := filepath.Join(filepath.Dir(filename), "..", "..", "..", "testdata", "local")
	dst := t.ArtifactDir()

	b, err := exec.Command("cp", "-a", src, dst).CombinedOutput()
	musta.NoErrorf(t, err, "%s", b)

	return filepath.Join(dst, "local")
}
