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

func NewFileStorage(originalDir string, processedDir string) *FileStorage {
	return &FileStorage{
		OriginalDir:  originalDir,
		ProcessedDir: processedDir,
	}
}

// SaveOriginal сохраняет оригинальный файл и возвращает полный путь.
func (fs *FileStorage) SaveOriginal(ctx context.Context, id string, filename string, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if err := os.MkdirAll(fs.OriginalDir, 0o755); err != nil {
		return "", err
	}

	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".bin"
	}
	ext = sanitizeExt(ext)

	fullPath := filepath.Join(fs.OriginalDir, id+ext)

	if err := writeFileAtomic(fullPath, data, 0o644); err != nil {
		return "", err
	}

	return fullPath, nil
}

// SaveProcessed сохраняет обработанный файл и возвращает полный путь.
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

// OpenProcessed открывает обработанный файл (для отдачи по GET /image/{id}).
func (fs *FileStorage) OpenProcessed(ctx context.Context, path string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// маленькая защита: не даём открывать файлы вне processed_dir
	if !isInsideDir(fs.ProcessedDir, path) {
		return nil, errors.New("processed path is outside storage dir")
	}

	return os.Open(path)
}

// OpenOriginal открывает оригинальный файл (для воркера).
func (fs *FileStorage) OpenOriginal(ctx context.Context, path string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// защита: не даём открывать файлы вне original_dir
	if !isInsideDir(fs.OriginalDir, path) {
		return nil, errors.New("original path is outside storage dir")
	}

	return os.Open(path)
}

// DeleteOriginal удаляет оригинальный файл.
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

// DeleteProcessed удаляет обработанный файл.
func (fs *FileStorage) DeleteProcessed(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !isInsideDir(fs.ProcessedDir, path) {
		return errors.New("processed path is outside storage dir")
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// --- helpers ---

func sanitizeExt(ext string) string {
	ext = strings.ToLower(ext)

	// оставим только безопасные расширения
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bin":
		return ext
	default:
		return ".bin"
	}
}

// writeFileAtomic пишет файл атомарно через temp + rename
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	// если что-то упало — удаляем temp
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

	// если rel начинается с ".." — значит файл вне директории
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

//TODO: CHECK
