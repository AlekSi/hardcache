//go:build unix

package local

import (
	"github.com/AlekSi/lazyerrors"
	"golang.org/x/sys/unix"
)

// DiskInfo returns the total and free disk space in bytes.
func DiskInfo(dir string) (total, free int64, err error) {
	var stat unix.Statfs_t
	if err = unix.Statfs(dir, &stat); err != nil {
		return -1, -1, lazyerrors.Error(err)
	}

	bsize := int64(stat.Bsize)
	total = bsize * int64(stat.Blocks)
	free = bsize * int64(stat.Bfree)
	return
}
