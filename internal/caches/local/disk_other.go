//go:build !unix && !windows

package local

import (
	"github.com/AlekSi/lazyerrors"
)

// DiskInfo is not implemented for this platform.
func DiskInfo(dir string) (total, free int64, err error) {
	return -1, -1, lazyerrors.New("local.DiskInfo: not implemented")
}
