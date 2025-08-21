package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileCopierNewFileCopier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		bufferSize  int64
		useZeroCopy bool
		wantBuffer  int64
	}{
		{
			name:        "positive buffer size",
			bufferSize:  1024,
			useZeroCopy: true,
			wantBuffer:  1024,
		},
		{
			name:        "zero buffer size uses default",
			bufferSize:  0,
			useZeroCopy: false,
			wantBuffer:  64 * 1024, // 64KB default
		},
		{
			name:        "negative buffer size uses default",
			bufferSize:  -100,
			useZeroCopy: false,
			wantBuffer:  64 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			copier := NewFileCopier(tt.bufferSize, tt.useZeroCopy)

			if copier.bufferSize != tt.wantBuffer {
				t.Errorf("NewFileCopier() bufferSize = %v, want %v", copier.bufferSize, tt.wantBuffer)
			}

			if copier.useZeroCopy != tt.useZeroCopy {
				t.Errorf("NewFileCopier() useZeroCopy = %v, want %v", copier.useZeroCopy, tt.useZeroCopy)
			}

			// Check default settings
			if !copier.preservePerms {
				t.Errorf("NewFileCopier() should preserve permissions by default")
			}

			if !copier.preserveTimes {
				t.Errorf("NewFileCopier() should preserve times by default")
			}
		})
	}
}

func TestFileCopierCopyFile(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	// Create source file

	sourceFile := filepath.Join(tempDir, "source.txt")

	sourceContent := []byte("Hello, World! This is test content.")
	if err := os.WriteFile(sourceFile, sourceContent, 0o644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Set specific modification time
	modTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(sourceFile, modTime, modTime); err != nil {
		t.Fatalf("Failed to set source file time: %v", err)
	}

	destFile := filepath.Join(tempDir, "dest.txt")

	copier := NewFileCopier(1024, false)
	ctx := context.Background()

	err := copier.CopyFile(ctx, sourceFile, destFile)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify destination file exists
	if _, statErr := os.Stat(destFile); os.IsNotExist(statErr) {
		t.Fatalf("Destination file was not created")
	}

	// Verify content
	destContent, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(destContent) != string(sourceContent) {
		t.Errorf("Content mismatch: got %q, want %q", string(destContent), string(sourceContent))
	}

	// Verify file info
	sourceInfo, err := os.Stat(sourceFile)
	if err != nil {
		t.Fatalf("Failed to stat source file: %v", err)
	}

	destInfo, err := os.Stat(destFile)
	if err != nil {
		t.Fatalf("Failed to stat destination file: %v", err)
	}

	// Check size
	if destInfo.Size() != sourceInfo.Size() {
		t.Errorf("Size mismatch: got %d, want %d", destInfo.Size(), sourceInfo.Size())
	}

	// Check modification time (should be preserved)
	if !destInfo.ModTime().Equal(sourceInfo.ModTime()) {
		t.Errorf("ModTime not preserved: got %v, want %v", destInfo.ModTime(), sourceInfo.ModTime())
	}

	// Check permissions
	if destInfo.Mode() != sourceInfo.Mode() {
		t.Errorf("Permissions not preserved: got %v, want %v", destInfo.Mode(), sourceInfo.Mode())
	}
}

func TestFileCopierCopyDirectory(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	// Create source directory with specific permissions
	sourceDir := filepath.Join(tempDir, "source_dir")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	destDir := filepath.Join(tempDir, "dest_dir")

	copier := NewFileCopier(1024, false)
	ctx := context.Background()

	err := copier.CopyFile(ctx, sourceDir, destDir)
	if err != nil {
		t.Fatalf("CopyFile for directory failed: %v", err)
	}

	// Verify destination directory exists
	destInfo, err := os.Stat(destDir)
	if err != nil {
		t.Fatalf("Destination directory was not created: %v", err)
	}

	if !destInfo.IsDir() {
		t.Errorf("Destination is not a directory")
	}

	// Verify permissions
	sourceInfo, err := os.Stat(sourceDir)
	if err != nil {
		t.Fatalf("Failed to stat source directory: %v", err)
	}

	if destInfo.Mode() != sourceInfo.Mode() {
		t.Errorf("Directory permissions not preserved: got %v, want %v", destInfo.Mode(), sourceInfo.Mode())
	}
}

func TestFileCopierSettings(t *testing.T) {
	t.Parallel()

	copier := NewFileCopier(1024, false)

	// Test SetPreservePermissions
	copier.SetPreservePermissions(false)

	if copier.preservePerms {
		t.Errorf("SetPreservePermissions(false) failed")
	}

	copier.SetPreservePermissions(true)

	if !copier.preservePerms {
		t.Errorf("SetPreservePermissions(true) failed")
	}

	// Test SetPreserveTimes
	copier.SetPreserveTimes(false)

	if copier.preserveTimes {
		t.Errorf("SetPreserveTimes(false) failed")
	}

	copier.SetPreserveTimes(true)

	if !copier.preserveTimes {
		t.Errorf("SetPreserveTimes(true) failed")
	}

	// Test SetBufferSize
	copier.SetBufferSize(2048)

	if copier.bufferSize != 2048 {
		t.Errorf("SetBufferSize(2048) failed: got %d", copier.bufferSize)
	}

	// Test invalid buffer size (should be ignored)
	copier.SetBufferSize(-100)

	if copier.bufferSize != 2048 {
		t.Errorf("SetBufferSize(-100) should be ignored: got %d", copier.bufferSize)
	}

	copier.SetBufferSize(0)

	if copier.bufferSize != 2048 {
		t.Errorf("SetBufferSize(0) should be ignored: got %d", copier.bufferSize)
	}
}

func TestFileCopierContextCancellation(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	// Create large source file
	sourceFile := filepath.Join(tempDir, "large_source.txt")

	largeContent := make([]byte, 1024*1024) // 1MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	if err := os.WriteFile(sourceFile, largeContent, 0o644); err != nil {
		t.Fatalf("Failed to create large source file: %v", err)
	}

	destFile := filepath.Join(tempDir, "large_dest.txt")

	copier := NewFileCopier(1024, false) // Small buffer to slow down copy

	// Create context that will be cancelled quickly
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	err := copier.CopyFile(ctx, sourceFile, destFile)
	if err == nil {
		t.Errorf("Expected copy to be cancelled, but it completed")
	}

	// The destination file should be cleaned up on error
	if _, err := os.Stat(destFile); !os.IsNotExist(err) {
		t.Errorf("Destination file should be cleaned up on cancellation")
	}
}

func TestFileCopierNonExistentSource(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	copier := NewFileCopier(1024, false)
	ctx := context.Background()

	err := copier.CopyFile(ctx, "/nonexistent/file.txt", filepath.Join(tempDir, "dest.txt"))
	if err == nil {
		t.Errorf("Expected error when copying nonexistent file")
	}
}

func TestFileCopierDestinationDirectoryCreation(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	// Create source file
	sourceFile := filepath.Join(tempDir, "source.txt")
	if err := os.WriteFile(sourceFile, []byte("content"), 0o644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Destination in nested directory that doesn't exist
	destFile := filepath.Join(tempDir, "nested", "subdir", "dest.txt")

	copier := NewFileCopier(1024, false)
	ctx := context.Background()

	err := copier.CopyFile(ctx, sourceFile, destFile)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify destination file exists
	if _, statErr := os.Stat(destFile); os.IsNotExist(statErr) {
		t.Fatalf("Destination file was not created")
	}

	// Verify content
	content, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(content) != "content" {
		t.Errorf("Content mismatch: got %q, want %q", string(content), "content")
	}
}
