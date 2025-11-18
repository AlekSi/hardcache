package local

import (
	"github.com/AlekSi/lazyerrors"
	"golang.org/x/sys/windows"
)

// DiskInfo returns the total and free disk space in bytes.
func DiskInfo(dir string) (total, free int64, err error) {
	directoryName, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return -1, -1, lazyerrors.Error(err)
	}

	var freeBytesAvailableToCaller uint64
	var totalNumberOfBytes uint64
	var totalNumberOfFreeBytes uint64
	err = windows.GetDiskFreeSpaceEx(directoryName, &freeBytesAvailableToCaller, &totalNumberOfBytes, &totalNumberOfFreeBytes)
	if err != nil {
		return -1, -1, lazyerrors.Error(err)
	}

	total = int64(totalNumberOfBytes)
	free = int64(totalNumberOfFreeBytes)
	return
}
