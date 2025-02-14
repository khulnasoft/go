package vfsutil

import (
	"archive/zip"
	"context"
	"io"
	"os"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/khulnasoft/go/go-langserver/diskcache"
)

// ArchiveCacheDir is the location on disk that archives are cached. It is
// configurable so that in production we can point it into CACHE_DIR.
var ArchiveCacheDir = "/tmp/go-langserver-archive-cache"

// MaxCacheSizeBytes is the maximum size of the cache directory after evicting
// entries. Defaults to 50 GB.
var MaxCacheSizeBytes = int64(50 * 1024 * 1024 * 1024)

// Evicter implements Evict
type Evicter interface {
	// Evict evicts an item from a cache.
	Evict()
}

type cachedFile struct {
	// File is an open FD to the fetched data
	File *os.File

	// path is the disk path for File
	path string
}

// Evict will remove the file from the cache. It does not close File. It also
// does not protect against other open readers or concurrent fetches.
func (f *cachedFile) Evict() {
	// Best-effort. Ignore error
	_ = os.Remove(f.path)
	cachedFileEvict.Inc()
}

// cachedFetch will open a file from the local cache with key. If missing,
// fetcher will fill the cache first. cachedFetch also performs
// single-flighting.
func cachedFetch(ctx context.Context, key string, s *diskcache.Store, fetcher func(context.Context) (io.ReadCloser, error)) (ff *cachedFile, err error) {
	f, err := s.Open(ctx, key, fetcher)
	if err != nil {
		return nil, err
	}
	return &cachedFile{
		File: f.File,
		path: f.Path,
	}, nil
}

func zipNewFileReader(f *os.File) (*zip.Reader, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return zip.NewReader(f, fi.Size())
}

var cachedFileEvict = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "golangserver_vfs_cached_file_evict",
	Help: "Total number of evictions to cachedFetch archives.",
})

func init() {
	prometheus.MustRegister(cachedFileEvict)
}
