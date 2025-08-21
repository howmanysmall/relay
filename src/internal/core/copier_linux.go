//go:build linux

package core

import (
	"context"
	"os"
	"syscall"
)

func (fc *FileCopier) zeroCopyLinux(ctx context.Context, src, dst *os.File, size int64) (int64, error) {
	var totalBytes int64

	chunkSize := int64(1024 * 1024) // 1MB chunks

	for totalBytes < size {
		select {
		case <-ctx.Done():
			return totalBytes, ctx.Err()
		default:
		}

		remaining := size - totalBytes
		if remaining < chunkSize {
			chunkSize = remaining
		}

		bytesWritten, err := syscall.Sendfile(int(dst.Fd()), int(src.Fd()), nil, int(chunkSize))
		if err != nil {
			return fc.bufferedCopy(ctx, src, dst)
		}

		totalBytes += int64(bytesWritten)

		if bytesWritten == 0 {
			break
		}
	}

	return totalBytes, nil
}