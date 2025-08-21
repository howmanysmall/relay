package core

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/howmanysmall/relay/src/internal/config"
)

// ConflictResolver handles file conflicts during synchronization.
type ConflictResolver struct {
	strategy    config.ConflictStrategy
	backup      bool
	backupDir   string
	interactive bool
}

// ConflictInfo contains information about a file conflict.
type ConflictInfo struct {
	Path       string
	SourceInfo *FileInfo
	DestInfo   *FileInfo
	Conflict   ConflictType
}

// ConflictType represents the type of file conflict.
type ConflictType int

// Conflict types
const (
	ConflictSizesDiffer ConflictType = iota
	ConflictModTimesDiffer
	ConflictChecksumsDiffer
	ConflictBothModified
)

func (ct ConflictType) String() string {
	switch ct {
	case ConflictSizesDiffer:
		return "file sizes differ"
	case ConflictModTimesDiffer:
		return "modification times differ"
	case ConflictChecksumsDiffer:
		return "checksums differ"
	case ConflictBothModified:
		return "both files modified"
	default:
		return "unknown conflict"
	}
}

// ConflictResolution represents how a conflict should be resolved.
type ConflictResolution int

// Conflict resolution options
const (
	ResolutionUseSource ConflictResolution = iota
	ResolutionUseDestination
	ResolutionSkip
	ResolutionBackupAndUseSource
	ResolutionMerge
)

// NewConflictResolver creates a new ConflictResolver using the provided configuration.
func NewConflictResolver(cfg *config.ConflictConfig) *ConflictResolver {
	if cfg == nil {
		cfg = &config.ConflictConfig{
			Strategy:    string(config.ConflictNewest),
			Backup:      false,
			Interactive: false,
		}
	}

	backupDir := cfg.BackupDir
	if backupDir == "" {
		backupDir = ".relay-backups"
	}

	return &ConflictResolver{
		strategy:    config.ConflictStrategy(cfg.Strategy),
		backup:      cfg.Backup,
		backupDir:   backupDir,
		interactive: cfg.Interactive,
	}
}

// ResolveConflict resolves a file conflict according to the configured strategy.
func (cr *ConflictResolver) ResolveConflict(_ context.Context, conflict *ConflictInfo) (ConflictResolution, error) {
	if cr.interactive {
		return cr.resolveInteractively(conflict)
	}

	switch cr.strategy {
	case config.ConflictNewest:
		return cr.resolveByNewest(conflict), nil
	case config.ConflictSource:
		return ResolutionUseSource, nil
	case config.ConflictDestination:
		return ResolutionUseDestination, nil
	case config.ConflictSmart:
		return cr.resolveSmart(conflict), nil
	case config.ConflictSkip:
		return ResolutionSkip, nil
	default:
		return cr.resolveByNewest(conflict), nil
	}
}

func (cr *ConflictResolver) resolveByNewest(conflict *ConflictInfo) ConflictResolution {
	if conflict.SourceInfo.ModTime.After(conflict.DestInfo.ModTime) {
		return ResolutionUseSource
	} else if conflict.DestInfo.ModTime.After(conflict.SourceInfo.ModTime) {
		return ResolutionUseDestination
	}

	// If times are equal, prefer source (safer for mirror operations)
	return ResolutionUseSource
}

func (cr *ConflictResolver) resolveSmart(conflict *ConflictInfo) ConflictResolution {
	// Smart resolution logic
	sizeDiff := conflict.SourceInfo.Size - conflict.DestInfo.Size
	timeDiff := conflict.SourceInfo.ModTime.Sub(conflict.DestInfo.ModTime)

	// If source is significantly newer (more than 1 minute), use source
	if timeDiff > time.Minute {
		return ResolutionUseSource
	}

	// If destination is significantly newer, use destination
	if timeDiff < -time.Minute {
		return ResolutionUseDestination
	}

	// If one file is much larger, it might be more complete
	if sizeDiff > 1024 { // Source is at least 1KB larger
		return ResolutionUseSource
	} else if sizeDiff < -1024 { // Destination is at least 1KB larger
		return ResolutionUseDestination
	}

	// Fall back to newest
	return cr.resolveByNewest(conflict)
}

func (cr *ConflictResolver) resolveInteractively(conflict *ConflictInfo) (ConflictResolution, error) {
	fmt.Printf("\n🔄 Conflict detected: %s\n", conflict.Path)
	fmt.Printf("Reason: %s\n\n", conflict.Conflict.String())

	fmt.Printf("Source file:\n")
	fmt.Printf("  Size: %d bytes\n", conflict.SourceInfo.Size)
	fmt.Printf("  Modified: %s\n", conflict.SourceInfo.ModTime.Format("2006-01-02 15:04:05"))

	if conflict.SourceInfo.Checksum != "" {
		fmt.Printf("  Checksum: %s\n", conflict.SourceInfo.Checksum[:16]+"...")
	}

	fmt.Printf("\nDestination file:\n")
	fmt.Printf("  Size: %d bytes\n", conflict.DestInfo.Size)
	fmt.Printf("  Modified: %s\n", conflict.DestInfo.ModTime.Format("2006-01-02 15:04:05"))

	if conflict.DestInfo.Checksum != "" {
		fmt.Printf("  Checksum: %s\n", conflict.DestInfo.Checksum[:16]+"...")
	}

	fmt.Printf("\nChoose resolution:\n")
	fmt.Printf("  [s] Use source (overwrite destination)\n")
	fmt.Printf("  [d] Use destination (keep current)\n")
	fmt.Printf("  [b] Backup destination and use source\n")
	fmt.Printf("  [k] Skip this file\n")
	fmt.Printf("  [v] View diff (if text files)\n")
	fmt.Printf("  [a] Apply to all similar conflicts\n")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\nYour choice [s/d/b/k/v/a]: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return ResolutionSkip, fmt.Errorf("failed to read input: %w", err)
		}

		choice := strings.ToLower(strings.TrimSpace(input))
		switch choice {
		case "s", "source":
			return ResolutionUseSource, nil
		case "d", "dest", "destination":
			return ResolutionUseDestination, nil
		case "b", "backup":
			return ResolutionBackupAndUseSource, nil
		case "k", "skip":
			return ResolutionSkip, nil
		case "v", "view", "diff":
			cr.showDiff(conflict)
			continue
		case "a", "all":
			return cr.promptForDefaultStrategy()
		default:
			fmt.Printf("Invalid choice. Please enter s, d, b, k, v, or a.\n")
			continue
		}
	}
}

func (cr *ConflictResolver) promptForDefaultStrategy() (ConflictResolution, error) {
	fmt.Printf("\nChoose default strategy for remaining conflicts:\n")
	fmt.Printf("  [s] Always use source\n")
	fmt.Printf("  [d] Always use destination\n")
	fmt.Printf("  [n] Always use newest\n")
	fmt.Printf("  [k] Always skip\n")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\nDefault strategy [s/d/n/k]: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return ResolutionSkip, fmt.Errorf("failed to read input: %w", err)
		}

		choice := strings.ToLower(strings.TrimSpace(input))
		switch choice {
		case "s", "source":
			cr.strategy = config.ConflictSource
			cr.interactive = false

			return ResolutionUseSource, nil
		case "d", "dest", "destination":
			cr.strategy = config.ConflictDestination
			cr.interactive = false

			return ResolutionUseDestination, nil
		case "n", "newest":
			cr.strategy = config.ConflictNewest
			cr.interactive = false

			return cr.resolveByNewest(&ConflictInfo{}), nil
		case "k", "skip":
			cr.strategy = config.ConflictSkip
			cr.interactive = false

			return ResolutionSkip, nil
		default:
			fmt.Printf("Invalid choice. Please enter s, d, n, or k.\n")
			continue
		}
	}
}

func (cr *ConflictResolver) showDiff(conflict *ConflictInfo) {
	// Simple diff for text files
	if conflict.SourceInfo.Size > 1024*1024 || conflict.DestInfo.Size > 1024*1024 {
		fmt.Printf("Files too large to diff (>1MB)\n")
		return
	}

	fmt.Printf("\nFile contents preview:\n")
	fmt.Printf("=== Source ===\n")

	if err := cr.showFilePreview(conflict.SourceInfo.Path, 10); err != nil {
		fmt.Printf("Error reading source: %v\n", err)
	}

	fmt.Printf("\n=== Destination ===\n")

	if err := cr.showFilePreview(conflict.DestInfo.Path, 10); err != nil {
		fmt.Printf("Error reading destination: %v\n", err)
	}
}

func (cr *ConflictResolver) showFilePreview(path string, lines int) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := file.Close(); cerr != nil {
			_ = cerr // ignore close error
		}
	}()

	scanner := bufio.NewScanner(file)

	lineCount := 0
	for scanner.Scan() && lineCount < lines {
		fmt.Printf("%d: %s\n", lineCount+1, scanner.Text())
		lineCount++
	}

	if lineCount == lines {
		fmt.Printf("... (truncated)\n")
	}

	return scanner.Err()
}

// CreateBackup creates a backup copy of the specified file if backups are enabled.
func (cr *ConflictResolver) CreateBackup(filePath string) (string, error) {
	if !cr.backup {
		return "", nil
	}

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(cr.backupDir, 0o750); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate unique backup filename
	fileName := filepath.Base(filePath)
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("%s.%s.backup", fileName, timestamp)
	backupPath := filepath.Join(cr.backupDir, backupName)

	// Copy file to backup location
	copier := NewFileCopier(0, false) // Use buffered copy for backups
	if err := copier.CopyFile(context.Background(), filePath, backupPath); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return backupPath, nil
}

// DetectConflict determines if a conflict exists between two file versions and returns details.
func (cr *ConflictResolver) DetectConflict(source, dest *FileInfo) *ConflictInfo {
	if source == nil || dest == nil {
		return nil
	}

	var conflictType ConflictType

	hasConflict := false

	// Check size differences
	switch {
	case source.Size != dest.Size:
		conflictType = ConflictSizesDiffer
		hasConflict = true
	case !source.ModTime.Equal(dest.ModTime):
		conflictType = ConflictModTimesDiffer
		hasConflict = true
	case source.Checksum != "" && dest.Checksum != "" && source.Checksum != dest.Checksum:
		conflictType = ConflictChecksumsDiffer
		hasConflict = true
	}

	if !hasConflict {
		return nil
	}

	return &ConflictInfo{
		Path:       source.Path,
		SourceInfo: source,
		DestInfo:   dest,
		Conflict:   conflictType,
	}
}
