// Package pack provides functionality for packing and unpacking directories.
package pack

import (
	"archive/zip"
	"compress/flate"
	"errors"
	"io"
	"io/fs"
	"os"

	"github.com/AlekSi/lazyerrors"
)

// Pack compresses the contents of the specified directory
// and writes it to the provided writer in ZIP format with the given comment.
func Pack(dir, comment string, w io.Writer) (resErr error) {
	zw := zip.NewWriter(w)
	defer func() {
		if e := zw.Close(); resErr == nil {
			resErr = lazyerrors.Error(e)
		}
	}()

	zw.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(w, flate.BestCompression)
	})

	root, err := os.OpenRoot(dir)
	if err != nil {
		resErr = lazyerrors.Error(err)
		return
	}

	defer func() {
		if e := root.Close(); resErr == nil {
			resErr = lazyerrors.Error(e)
		}
	}()

	if err = zw.AddFS(root.FS()); err != nil {
		resErr = lazyerrors.Error(err)
		return
	}

	if err = zw.SetComment(comment); err != nil {
		resErr = lazyerrors.Error(err)
		return
	}

	return
}

// putFile writes the contents of src file to the dst path.
// If dst already exists, it will be touched, but not overwritten.
func putFile(dst string, src fs.File) (resErr error) {
	srcFI, err := src.Stat()
	if err != nil {
		resErr = lazyerrors.Error(err)
		return
	}

	dstFI, err := os.Stat(dst)
	if err == nil {
		if err = os.Chtimes(dst, dstFI.ModTime(), srcFI.ModTime()); err != nil {
			resErr = lazyerrors.Error(err)
		}

		return
	}

	if !errors.Is(err, fs.ErrNotExist) {
		resErr = lazyerrors.Error(err)
		return
	}

	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, srcFI.Mode())
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			// another process created the file in the meantime
			return
		}

		resErr = lazyerrors.Error(err)
		return
	}

	defer func() {
		if e := f.Close(); resErr == nil {
			resErr = lazyerrors.Error(e)
		}
	}()

	if _, err = io.Copy(f, src); err != nil {
		resErr = lazyerrors.Error(err)
		return
	}

	return
}

// Unpack extracts the contents of a ZIP archive from the provided reader
// and writes it to the specified directory.
// It also returns the archive comment.
func Unpack(r io.ReaderAt, size int64, dir string) (comment string, resErr error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		resErr = lazyerrors.Error(err)
		return
	}

	comment = zr.Comment

	root, err := os.OpenRoot(dir)
	if err != nil {
		resErr = lazyerrors.Error(err)
		return
	}
	defer func() {
		if e := root.Close(); resErr == nil {
			resErr = lazyerrors.Error(e)
		}
	}()

	for _, f := range zr.File {
		fp := f.Name

		if f.FileInfo().IsDir() {
			err = root.MkdirAll(fp, f.Mode())
			if err != nil {
				resErr = lazyerrors.Error(err)
				return
			}

			continue
		}

		// err = root.MkdirAll(fp[:len(fp)-len(f.FileInfo().Name())], 0o755)
		// if err != nil {
		// 	resErr = lazyerrors.Error(err)
		// 	return
		// }

		dst, err := root.OpenFile(fp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			resErr = lazyerrors.Error(err)
			return
		}

		rc, err := f.Open()
		if err != nil {
			dst.Close()
			resErr = lazyerrors.Error(err)
			return
		}

		_, err = io.Copy(dst, rc)
		rc.Close()
		dst.Close()
		if err != nil {
			resErr = lazyerrors.Error(err)
			return
		}
	}

	return
}
