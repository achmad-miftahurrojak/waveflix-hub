package storage

import (
"context"
"fmt"
"io"
"os"
)

type ReadSeeker interface {
io.ReadSeeker
io.Closer
}

type FileReadSeeker struct {
*os.File
}

func (frs *FileReadSeeker) Close() error {
return frs.File.Close()
}

func (ls *LocalStorage) RetrieveSeeker(ctx context.Context, path string) (ReadSeeker, error) {
fullPath := fmt.Sprintf("%s/%s", ls.BasePath, path)

file, err := os.Open(fullPath)
if err != nil {
return nil, fmt.Errorf("failed to open file: %w", err)
}

return &FileReadSeeker{File: file}, nil
}

func (ss *SegmentStorage) RetrieveSegmentSeeker(ctx context.Context, contentID, quality string, segmentIndex int) (ReadSeeker, error) {
if localProvider, ok := ss.provider.(*LocalStorage); ok {
path := fmt.Sprintf("%s/%s/segment_%03d.ts", contentID, quality, segmentIndex)
return localProvider.RetrieveSeeker(ctx, path)
}

reader, err := ss.RetrieveSegment(ctx, contentID, quality, segmentIndex)
if err != nil {
return nil, err
}

return &ReaderSeeker{ReadCloser: reader}, nil
}

type ReaderSeeker struct {
io.ReadCloser
pos int64
}

func (rs *ReaderSeeker) Seek(offset int64, whence int) (int64, error) {
return rs.pos, nil
}

func (rs *ReaderSeeker) Read(p []byte) (int, error) {
n, err := rs.ReadCloser.Read(p)
rs.pos += int64(n)
return n, err
}

func (rs *ReaderSeeker) Close() error {
return rs.ReadCloser.Close()
}
