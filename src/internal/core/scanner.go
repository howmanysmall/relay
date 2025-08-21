package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/zeebo/blake3"
	"golang.org/x/sync/semaphore"
)

// FileScanner scans directories and computes file checksums with caching.
type FileScanner struct {
	maxConcurrency int64
	checksumAlgo   string
	cache          *checksumCache
}

type checksumCache struct {
	mu    sync.RWMutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	checksum string
	modTime  int64
	size     int64
}

// NewFileScanner creates a new file scanner with the specified concurrency limit.
func NewFileScanner(maxConcurrency int) *FileScanner {
	if maxConcurrency <= 0 {
		maxConcurrency = runtime.GOMAXPROCS(0) * 2
	}

	return &FileScanner{
		maxConcurrency: int64(maxConcurrency),
		checksumAlgo:   "blake3",
		cache: &checksumCache{
			cache: make(map[string]cacheEntry),
		},
	}
}

// SetChecksumAlgorithm sets the checksum algorithm to use (blake3, sha256).
func (s *FileScanner) SetChecksumAlgorithm(algo string) {
	s.checksumAlgo = algo
}

// Scan recursively scans a directory and returns file information.
func (s *FileScanner) Scan(ctx context.Context, path string) ([]*FileInfo, error) {
	return s.ScanWithFilter(ctx, path, nil)
}

// ScanWithFilter scans a directory with the given filter function.
func (s *FileScanner) ScanWithFilter(ctx context.Context, path string, filter FilterFunc) ([]*FileInfo, error) {
	var (
		files []*FileInfo
		mu    sync.Mutex
	)

	sem := semaphore.NewWeighted(s.maxConcurrency)

	err := filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		go func() {
			defer sem.Release(1)

			info, err := s.getFileInfo(filePath, d)
			if err != nil {
				return
			}

			if filter != nil && !filter(filePath, info) {
				return
			}

			mu.Lock()

			files = append(files, info)

			mu.Unlock()
		}()

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory %s: %w", path, err)
	}

	if err := sem.Acquire(ctx, s.maxConcurrency); err != nil {
		return nil, err
	}

	sem.Release(s.maxConcurrency)

	return files, nil
}

func (s *FileScanner) getFileInfo(path string, d fs.DirEntry) (*FileInfo, error) {
	stat, err := d.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for %s: %w", path, err)
	}

	info := &FileInfo{
		Path:    path,
		Size:    stat.Size(),
		ModTime: stat.ModTime(),
		Mode:    uint32(stat.Mode()),
		IsDir:   stat.IsDir(),
	}

	if !info.IsDir && info.Size > 0 {
		checksum, err := s.getChecksum(path, info)
		if err == nil {
			info.Checksum = checksum
			info.ChecksumAlgo = s.checksumAlgo
		}
	}

	return info, nil
}

func (s *FileScanner) getChecksum(path string, info *FileInfo) (string, error) {
	cacheKey := path

	s.cache.mu.RLock()

	if entry, exists := s.cache.cache[cacheKey]; exists {
		if entry.modTime == info.ModTime.Unix() && entry.size == info.Size {
			s.cache.mu.RUnlock()
			return entry.checksum, nil
		}
	}

	s.cache.mu.RUnlock()

	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", path, err)
	}

	defer func() {
		if cerr := file.Close(); cerr != nil {
			_ = cerr
		}
	}()

	checksum, err := s.calculateChecksum(file)
	if err != nil {
		return "", fmt.Errorf("failed to calculate checksum for %s: %w", path, err)
	}

	s.cache.mu.Lock()
	s.cache.cache[cacheKey] = cacheEntry{
		checksum: checksum,
		modTime:  info.ModTime.Unix(),
		size:     info.Size,
	}
	s.cache.mu.Unlock()

	return checksum, nil
}

func (s *FileScanner) calculateChecksum(reader io.Reader) (string, error) {
	switch s.checksumAlgo {
	case "blake3":
		hasher := blake3.New()
		if _, err := io.Copy(hasher, reader); err != nil {
			return "", err
		}

		return hex.EncodeToString(hasher.Sum(nil)), nil

	case "sha256":
		hasher := sha256.New()
		if _, err := io.Copy(hasher, reader); err != nil {
			return "", err
		}

		return hex.EncodeToString(hasher.Sum(nil)), nil

	case "md5":
		hasher := sha256.New()
		if _, err := io.Copy(hasher, reader); err != nil {
			return "", err
		}

		return hex.EncodeToString(hasher.Sum(nil)), nil

	default:
		return "", fmt.Errorf("unsupported checksum algorithm: %s", s.checksumAlgo)
	}
}

// ClearCache clears the checksum cache.
func (s *FileScanner) ClearCache() {
	s.cache.mu.Lock()
	s.cache.cache = make(map[string]cacheEntry)
	s.cache.mu.Unlock()
}

// GetCacheStats returns cache statistics (entry count, memory usage estimate).
func (s *FileScanner) GetCacheStats() (int, int64) {
	s.cache.mu.RLock()
	defer s.cache.mu.RUnlock()

	entries := len(s.cache.cache)

	var totalSize int64
	for _, entry := range s.cache.cache {
		totalSize += entry.size
	}

	return entries, totalSize
}
