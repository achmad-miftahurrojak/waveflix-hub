package storage

import (
"context"
"fmt"
"io"
"os"
"path/filepath"
)

type StorageProvider interface {
Store(ctx context.Context, path string, data io.Reader) error
Retrieve(ctx context.Context, path string) (io.ReadCloser, error)
Delete(ctx context.Context, path string) error
Exists(ctx context.Context, path string) (bool, error)
List(ctx context.Context, prefix string) ([]string, error)
}

type LocalStorage struct {
BasePath string
}

type S3Storage struct {
Bucket    string
Region    string
AccessKey string
SecretKey string
}

func NewLocalStorage(basePath string) *LocalStorage {
return &LocalStorage{BasePath: basePath}
}

func (ls *LocalStorage) Store(ctx context.Context, path string, data io.Reader) error {
fullPath := filepath.Join(ls.BasePath, path)

if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
return fmt.Errorf("failed to create directory: %w", err)
}

file, err := os.Create(fullPath)
if err != nil {
return fmt.Errorf("failed to create file: %w", err)
}
defer file.Close()

_, err = io.Copy(file, data)
if err != nil {
return fmt.Errorf("failed to write data: %w", err)
}

return nil
}

func (ls *LocalStorage) Retrieve(ctx context.Context, path string) (io.ReadCloser, error) {
fullPath := filepath.Join(ls.BasePath, path)

file, err := os.Open(fullPath)
if err != nil {
return nil, fmt.Errorf("failed to open file: %w", err)
}

return file, nil
}

func (ls *LocalStorage) Delete(ctx context.Context, path string) error {
fullPath := filepath.Join(ls.BasePath, path)

err := os.Remove(fullPath)
if err != nil && !os.IsNotExist(err) {
return fmt.Errorf("failed to delete file: %w", err)
}

return nil
}

func (ls *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
fullPath := filepath.Join(ls.BasePath, path)

_, err := os.Stat(fullPath)
if err != nil {
if os.IsNotExist(err) {
return false, nil
}
return false, fmt.Errorf("failed to check file existence: %w", err)
}

return true, nil
}

func (ls *LocalStorage) List(ctx context.Context, prefix string) ([]string, error) {
fullPrefix := filepath.Join(ls.BasePath, prefix)

var files []string
err := filepath.Walk(fullPrefix, func(path string, info os.FileInfo, err error) error {
if err != nil {
return err
}

if !info.IsDir() {
relPath, err := filepath.Rel(ls.BasePath, path)
if err != nil {
return err
}
files = append(files, filepath.ToSlash(relPath))
}

return nil
})

if err != nil && !os.IsNotExist(err) {
return nil, fmt.Errorf("failed to list files: %w", err)
}

return files, nil
}

type SegmentStorage struct {
provider StorageProvider
}

func NewSegmentStorage(provider StorageProvider) *SegmentStorage {
return &SegmentStorage{provider: provider}
}

func (ss *SegmentStorage) StoreSegment(ctx context.Context, contentID, quality string, segmentIndex int, data io.Reader) error {
path := fmt.Sprintf("%s/%s/segment_%03d.ts", contentID, quality, segmentIndex)
return ss.provider.Store(ctx, path, data)
}

func (ss *SegmentStorage) StorePlaylist(ctx context.Context, contentID, quality string, data io.Reader) error {
path := fmt.Sprintf("%s/%s/playlist.m3u8", contentID, quality)
return ss.provider.Store(ctx, path, data)
}

func (ss *SegmentStorage) StoreMasterPlaylist(ctx context.Context, contentID string, data io.Reader) error {
path := fmt.Sprintf("%s/master.m3u8", contentID)
return ss.provider.Store(ctx, path, data)
}

func (ss *SegmentStorage) RetrieveSegment(ctx context.Context, contentID, quality string, segmentIndex int) (io.ReadCloser, error) {
path := fmt.Sprintf("%s/%s/segment_%03d.ts", contentID, quality, segmentIndex)
return ss.provider.Retrieve(ctx, path)
}

func (ss *SegmentStorage) RetrievePlaylist(ctx context.Context, contentID, quality string) (io.ReadCloser, error) {
path := fmt.Sprintf("%s/%s/playlist.m3u8", contentID, quality)
return ss.provider.Retrieve(ctx, path)
}

func (ss *SegmentStorage) RetrieveMasterPlaylist(ctx context.Context, contentID string) (io.ReadCloser, error) {
path := fmt.Sprintf("%s/master.m3u8", contentID)
return ss.provider.Retrieve(ctx, path)
}

func (ss *SegmentStorage) DeleteContent(ctx context.Context, contentID string) error {
files, err := ss.provider.List(ctx, contentID)
if err != nil {
return err
}

for _, file := range files {
if err := ss.provider.Delete(ctx, file); err != nil {
return err
}
}

return nil
}

func (ss *SegmentStorage) ListSegments(ctx context.Context, contentID, quality string) ([]string, error) {
prefix := fmt.Sprintf("%s/%s/segment_", contentID, quality)
return ss.provider.List(ctx, prefix)
}
