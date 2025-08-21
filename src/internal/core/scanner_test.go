package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestFileScannerNewFileScanner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		maxConcurrency  int
		wantConcurrency int64
	}{
		{
			name:            "positive concurrency",
			maxConcurrency:  4,
			wantConcurrency: 4,
		},
		{
			name:            "zero concurrency uses default",
			maxConcurrency:  0,
			wantConcurrency: int64(defaultConcurrency()),
		},
		{
			name:            "negative concurrency uses default",
			maxConcurrency:  -1,
			wantConcurrency: int64(defaultConcurrency()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scanner := NewFileScanner(tt.maxConcurrency)
			if scanner.maxConcurrency != tt.wantConcurrency {
				t.Errorf("NewFileScanner() concurrency = %v, want %v", scanner.maxConcurrency, tt.wantConcurrency)
			}

			if scanner.checksumAlgo != "blake3" {
				t.Errorf("NewFileScanner() algorithm = %v, want blake3", scanner.checksumAlgo)
			}
		})
	}
}

func TestFileScannerSetChecksumAlgorithm(t *testing.T) {
	t.Parallel()

	scanner := NewFileScanner(1)

	tests := []string{"blake3", "sha256", "md5"}
	for _, algo := range tests {
		scanner.SetChecksumAlgorithm(algo)

		if scanner.checksumAlgo != algo {
			t.Errorf("SetChecksumAlgorithm(%s) failed, got %s", algo, scanner.checksumAlgo)
		}
	}
}

func TestFileScannerScan(t *testing.T) {
	t.Parallel()
	// Create temporary test directory
	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string][]byte{
		"file1.txt":        []byte("Hello, World!"),
		"file2.txt":        []byte("Test content"),
		"subdir/file3.txt": []byte("Subdirectory file"),
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		if err := os.WriteFile(fullPath, content, 0o644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", path, err)
		}
	}

	scanner := NewFileScanner(2)
	ctx := context.Background()

	files, err := scanner.Scan(ctx, tempDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should find 3 files + 1 subdirectory (might include root dir too)
	expectedMin := 4
	if len(files) < expectedMin {
		t.Errorf("Expected at least %d items, got %d", expectedMin, len(files))
	}

	// Verify file properties

	fileMap := make(map[string]*FileInfo)

	for _, file := range files {
		relPath, _ := filepath.Rel(tempDir, file.Path)
		fileMap[relPath] = file
	}

	// Check regular files
	for path, expectedContent := range testFiles {
		info, exists := fileMap[path]
		if !exists {
			t.Errorf("File %s not found in scan results", path)
			continue
		}

		if info.IsDir {
			t.Errorf("File %s incorrectly marked as directory", path)
		}

		if info.Size != int64(len(expectedContent)) {
			t.Errorf("File %s size = %d, want %d", path, info.Size, len(expectedContent))
		}

		if info.Checksum == "" {
			t.Errorf("File %s missing checksum", path)
		}
	}

	// Check subdirectory
	if info, exists := fileMap["subdir"]; exists {
		if !info.IsDir {
			t.Errorf("Subdirectory not marked as directory")
		}
	}
}

func TestFileScannerScanWithFilter(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	// Create test files
	files := []string{"test.txt", "test.log", "readme.md", "config.json"}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(tempDir, file), []byte("content"), 0o644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	scanner := NewFileScanner(1)
	ctx := context.Background()

	// Filter for .txt and .md files only
	filter := func(path string, _ *FileInfo) bool {
		ext := filepath.Ext(path)
		return ext == ".txt" || ext == ".md"
	}

	results, err := scanner.ScanWithFilter(ctx, tempDir, filter)
	if err != nil {
		t.Fatalf("ScanWithFilter failed: %v", err)
	}

	// Should find 2 files (test.txt and readme.md)
	if len(results) != 2 {
		t.Errorf("Expected 2 filtered files, got %d", len(results))
	}

	// Verify filtered files
	foundFiles := make(map[string]bool)
	for _, result := range results {
		foundFiles[filepath.Base(result.Path)] = true
	}

	if !foundFiles["test.txt"] || !foundFiles["readme.md"] {
		t.Errorf("Filter did not return expected files")
	}
}

func TestFileScannerCacheStats(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	scanner := NewFileScanner(1)
	ctx := context.Background()

	// Initial cache should be empty

	count, size := scanner.GetCacheStats()
	if count != 0 || size != 0 {
		t.Errorf("Initial cache stats = (%d, %d), want (0, 0)", count, size)
	}

	// Scan to populate cache
	_, err := scanner.Scan(ctx, tempDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Cache should now have entries
	count, _ = scanner.GetCacheStats()
	if count == 0 {
		t.Errorf("Cache should have entries after scan")
	}

	// Clear cache
	scanner.ClearCache()

	count, size = scanner.GetCacheStats()
	if count != 0 || size != 0 {
		t.Errorf("Cache stats after clear = (%d, %d), want (0, 0)", count, size)
	}
}

func TestFileScannerContextCancellation(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	// Create many files to ensure scan takes time
	for i := 0; i < 100; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(filename, []byte("content"), 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	scanner := NewFileScanner(1) // Low concurrency to slow down scan

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := scanner.Scan(ctx, tempDir)
	if err == nil {
		t.Errorf("Expected scan to be cancelled, but it completed")
	}

	// Check if error is related to context cancellation
	if !errors.Is(err, context.DeadlineExceeded) && !isContextCancelError(err) {
		t.Errorf("Expected context cancellation error, got %v", err)
	}
}

func TestFileScannerNonExistentDirectory(t *testing.T) {
	t.Parallel()

	scanner := NewFileScanner(1)
	ctx := context.Background()

	_, err := scanner.Scan(ctx, "/nonexistent/directory")
	if err == nil {
		t.Errorf("Expected error when scanning nonexistent directory")
	}
}

// Helper function to get default concurrency
func defaultConcurrency() int {
	// This should match the logic in NewFileScanner
	return runtime.GOMAXPROCS(0) * 2
}

// Helper function to check if error is context-related
func isContextCancelError(err error) bool {
	if err == nil {
		return false
	}
	// Check if error message contains context-related keywords
	errStr := err.Error()

	return strings.Contains(errStr, "context") ||
		strings.Contains(errStr, "deadline") ||
		strings.Contains(errStr, "cancelled")
}
