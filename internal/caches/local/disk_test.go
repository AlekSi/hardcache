package local

import (
	"os/exec"
	"testing"

	"github.com/AlekSi/shoulda"
	"github.com/AlekSi/shoulda/cmp"
	"github.com/stretchr/testify/require"
)

func TestDiskInfo(t *testing.T) {
	b, err := exec.Command("df", "-m", "/").CombinedOutput()
	require.NoError(t, err)
	t.Logf("\n%s", b)

	total, free, err := DiskInfo("/")
	require.NoError(t, err)

	t.Logf("\n           total: %dM,    free: %dM", total/1024/1024, free/1024/1024)

	shoulda.ReturnTrue(t, total, cmp.Greater, 0)
	shoulda.ReturnTrue(t, free, cmp.Greater, 0)
	shoulda.ReturnTrue(t, total, cmp.Greater, free)
}
