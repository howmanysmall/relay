//go:build !linux

package core

import (
	"context"
	"os"
)

func (fc *FileCopier) zeroCopyLinux(ctx context.Context, src, dst *os.File, _ int64) (int64, error) {
	// On non-Linux platforms, fall back to buffered copy
	return fc.bufferedCopy(ctx, src, dst)
}
