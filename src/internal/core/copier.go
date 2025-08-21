// Package core provides the core file synchronization functionality.
package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// FileCopier handles copying files with various optimizations and options.
type FileCopier struct {
	bufferSize    int64
	useZeroCopy   bool
	preservePerms bool
	preserveTimes bool
	workers       int
}

// NewFileCopier creates a new file copier with the specified buffer size and zero-copy option.
func NewFileCopier(bufferSize int64, useZeroCopy bool) *FileCopier {
	if bufferSize <= 0 {
		bufferSize = 64 * 1024 // 64KB default
	}

	return &FileCopier{
		bufferSize:    bufferSize,
		useZeroCopy:   useZeroCopy,
		preservePerms: true,
		preserveTimes: true,
		workers:       runtime.GOMAXPROCS(0),
	}
}

// CopyFile copies a file or directory from source to destination.
func (fc *FileCopier) CopyFile(ctx context.Context, src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file %s: %w", src, err)
	}

	if srcInfo.IsDir() {
		return fc.copyDirectory(ctx, src, dst, srcInfo)
	}

	return fc.copyRegularFile(ctx, src, dst, srcInfo)
}

func (fc *FileCopier) copyRegularFile(ctx context.Context, src, dst string, srcInfo os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}

	defer func() {
		if err := dstFile.Close(); err != nil {
			_ = err // ignore close error
		}
	}()

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}

	defer func() {
		if err := srcFile.Close(); err != nil {
			_ = err // ignore close error
		}
	}()

	var bytesWritten int64
	if fc.useZeroCopy && fc.canUseZeroCopy(srcFile, dstFile) {
		bytesWritten, err = fc.zeroCopy(ctx, srcFile, dstFile, srcInfo.Size())
	} else {
		bytesWritten, err = fc.bufferedCopy(ctx, srcFile, dstFile)
	}

	if err != nil {
		if removeErr := os.Remove(dst); removeErr != nil {
			_ = removeErr
		}

		return fmt.Errorf("failed to copy file content: %w", err)
	}

	if bytesWritten != srcInfo.Size() {
		if removeErr := os.Remove(dst); removeErr != nil {
			_ = removeErr
		}

		return fmt.Errorf("incomplete copy: expected %d bytes, wrote %d bytes", srcInfo.Size(), bytesWritten)
	}

	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	if fc.preservePerms {
		if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
			return fmt.Errorf("failed to set file permissions: %w", err)
		}
	}

	if fc.preserveTimes {
		if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
			return fmt.Errorf("failed to set file times: %w", err)
		}
	}

	return nil
}

func (fc *FileCopier) copyDirectory(_ context.Context, _, dst string, srcInfo os.FileInfo) error {
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", dst, err)
	}

	if fc.preservePerms {
		if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
			return fmt.Errorf("failed to set directory permissions: %w", err)
		}
	}

	if fc.preserveTimes {
		if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
			return fmt.Errorf("failed to set directory times: %w", err)
		}
	}

	return nil
}

func (fc *FileCopier) bufferedCopy(ctx context.Context, src io.Reader, dst io.Writer) (int64, error) {
	buffer := make([]byte, fc.bufferSize)

	var totalBytes int64

	for {
		select {
		case <-ctx.Done():
			return totalBytes, ctx.Err()
		default:
		}

		bytesRead, err := src.Read(buffer)
		if bytesRead > 0 {
			bytesWritten, writeErr := dst.Write(buffer[:bytesRead])
			totalBytes += int64(bytesWritten)

			if writeErr != nil {
				return totalBytes, writeErr
			}

			if bytesWritten != bytesRead {
				return totalBytes, fmt.Errorf("short write: expected %d, wrote %d", bytesRead, bytesWritten)
			}
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return totalBytes, err
		}
	}

	return totalBytes, nil
}

func (fc *FileCopier) canUseZeroCopy(src, _ *os.File) bool {
	if !fc.useZeroCopy {
		return false
	}

	srcStat, err := src.Stat()
	if err != nil {
		return false
	}

	if srcStat.Mode()&os.ModeType != 0 {
		return false
	}

	return true
}

func (fc *FileCopier) zeroCopy(ctx context.Context, src, dst *os.File, size int64) (int64, error) {
	switch runtime.GOOS {
	case "linux":
		return fc.zeroCopyLinux(ctx, src, dst, size)
	case "darwin":
		return fc.zeroCopyDarwin(ctx, src, dst, size)
	default:
		return fc.bufferedCopy(ctx, src, dst)
	}
}

func (fc *FileCopier) zeroCopyDarwin(ctx context.Context, src, dst *os.File, _ int64) (int64, error) {
	// macOS copyfile is not available through standard syscalls in Go
	// Fall back to buffered copy for now
	return fc.bufferedCopy(ctx, src, dst)
}

// SetPreservePermissions sets whether to preserve file permissions during copy.
func (fc *FileCopier) SetPreservePermissions(preserve bool) {
	fc.preservePerms = preserve
}

// SetPreserveTimes sets whether to preserve file modification times during copy.
func (fc *FileCopier) SetPreserveTimes(preserve bool) {
	fc.preserveTimes = preserve
}

// SetBufferSize sets the buffer size for file operations.
func (fc *FileCopier) SetBufferSize(size int64) {
	if size > 0 {
		fc.bufferSize = size
	}
}
