package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FileStorage struct {
	OriginalDir  string
	ProcessedDir string
}

func NewFileStorage(originalDir, processedDir string) *FileStorage {
	return &FileStorage{OriginalDir: originalDir, ProcessedDir: processedDir}
}

func (fs *FileStorage) OpenOriginal(ctx context.Context, path string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !isInsideDir(fs.OriginalDir, path) {
		return nil, errors.New("original path is outside storage dir")
	}
	return os.Open(path)
}

func (fs *FileStorage) DeleteOriginal(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !isInsideDir(fs.OriginalDir, path) {
		return errors.New("original path is outside storage dir")
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (fs *FileStorage) SaveProcessed(ctx context.Context, id string, ext string, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(fs.ProcessedDir, 0o755); err != nil {
		return "", err
	}
	if ext == "" {
		ext = ".jpg"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	ext = sanitizeExt(ext)

	fullPath := filepath.Join(fs.ProcessedDir, id+ext)
	if err := writeFileAtomic(fullPath, data, 0o644); err != nil {
		return "", err
	}
	return fullPath, nil
}

func sanitizeExt(ext string) string {
	ext = strings.ToLower(ext)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bin":
		return ext
	default:
		return ".bin"
	}
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func isInsideDir(baseDir, filePath string) bool {
	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return false
	}
	fileAbs, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(baseAbs, fileAbs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
