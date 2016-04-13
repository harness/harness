package fscache

import (
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
	
	"gopkg.in/djherbis/stream.v1"
)

// Cache works like a concurrent-safe map for streams.
type Cache interface {

	// Get manages access to the streams in the cache.
	// If the key does not exist, w != nil and you can start writing to the stream.
	// If the key does exist, w == nil.
	// r will always be non-nil as long as err == nil and you must close r when you're done reading.
	// Get can be called concurrently, and writing and reading is concurrent safe.
	Get(key string) (ReadAtCloser, io.WriteCloser, error)

	// Remove deletes the stream from the cache, blocking until the underlying
	// file can be deleted (all active streams finish with it).
	// It is safe to call Remove concurrently with Get.
	Remove(key string) error

	// Exists checks if a key is in the cache.
	// It is safe to call Exists concurrently with Get.
	Exists(key string) bool

	// Clean will empty the cache and delete the cache folder.
	// Clean is not safe to call while streams are being read/written.
	Clean() error
}

type cache struct {
	mu    sync.RWMutex
	files map[string]fileStream
	grim  Reaper
	fs    FileSystem
}

// ReadAtCloser is an io.ReadCloser, and an io.ReaderAt. It supports both so that Range
// Requests are possible.
type ReadAtCloser interface {
	io.ReadCloser
	io.ReaderAt
}

type fileStream interface {
	next() (ReadAtCloser, error)
	inUse() bool
	io.WriteCloser
	Remove() error
	Name() string
}

// New creates a new Cache using NewFs(dir, perms).
// expiry is the duration after which an un-accessed key will be removed from
// the cache, a zero value expiro means never expire.
func New(dir string, perms os.FileMode, expiry time.Duration) (Cache, error) {
	fs, err := NewFs(dir, perms)
	if err != nil {
		return nil, err
	}
	var grim Reaper
	if expiry > 0 {
		grim = &reaper{
			expiry: expiry,
			period: expiry,
		}
	}
	return NewCache(fs, grim)
}

// NewCache creates a new Cache based on FileSystem fs.
// fs.Files() are loaded using the name they were created with as a key.
// Reaper is used to determine when files expire, nil means never expire.
func NewCache(fs FileSystem, grim Reaper) (Cache, error) {
	c := &cache{
		files: make(map[string]fileStream),
		grim:  grim,
		fs:    fs,
	}
	err := c.load()
	if err != nil {
		return nil, err
	}
	if grim != nil {
		c.haunter()
	}
	return c, nil
}

func (c *cache) haunter() {
	c.haunt()
	time.AfterFunc(c.grim.Next(), c.haunter)
}

func (c *cache) haunt() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, f := range c.files {
		if f.inUse() {
			continue
		}

		lastRead, lastWrite, err := c.fs.AccessTimes(f.Name())
		if err != nil {
			continue
		}

		if c.grim.Reap(key, lastRead, lastWrite) {
			delete(c.files, key)
			c.fs.Remove(f.Name())
		}
	}
	return
}

func (c *cache) load() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fs.Reload(func(key, name string) {
		c.files[key] = c.oldFile(name)
	})
}

func (c *cache) Exists(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.files[key]
	return ok
}

func (c *cache) Get(key string) (r ReadAtCloser, w io.WriteCloser, err error) {
	c.mu.RLock()
	f, ok := c.files[key]
	if ok {
		r, err = f.next()
		c.mu.RUnlock()
		return r, nil, err
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	f, ok = c.files[key]
	if ok {
		r, err = f.next()
		return r, nil, err
	}

	f, err = c.newFile(key)
	if err != nil {
		return nil, nil, err
	}

	r, err = f.next()
	if err != nil {
		f.Close()
		c.fs.Remove(f.Name())
		return nil, nil, err
	}

	c.files[key] = f

	return r, f, err
}

func (c *cache) Remove(key string) error {
	c.mu.Lock()
	f, ok := c.files[key]
	delete(c.files, key)
	c.mu.Unlock()

	if ok {
		return f.Remove()
	}
	return nil
}

func (c *cache) Clean() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files = make(map[string]fileStream)
	return c.fs.RemoveAll()
}

type cachedFile struct {
	stream *stream.Stream
	handleCounter
}

func (c *cache) newFile(name string) (fileStream, error) {
	s, err := stream.NewStream(name, c.fs)
	if err != nil {
		return nil, err
	}
	cf := &cachedFile{
		stream: s,
	}
	cf.inc()
	return cf, nil
}

func (c *cache) oldFile(name string) fileStream {
	return &reloadedFile{
		fs:   c.fs,
		name: name,
	}
}

type reloadedFile struct {
	fs   FileSystem
	name string
	handleCounter
	io.WriteCloser // nop Write & Close methods. will never be called.
}

func (f *reloadedFile) Name() string { return f.name }

func (f *reloadedFile) Remove() error {
	f.waitUntilFree()
	return f.fs.Remove(f.name)
}

func (f *reloadedFile) next() (r ReadAtCloser, err error) {
	r, err = f.fs.Open(f.name)
	if err == nil {
		f.inc()
	}
	return &cacheReader{r: r, cnt: &f.handleCounter}, err
}

func (f *cachedFile) Name() string { return f.stream.Name() }

func (f *cachedFile) Remove() error { return f.stream.Remove() }

func (f *cachedFile) next() (r ReadAtCloser, err error) {
	reader, err := f.stream.NextReader()
	if err != nil {
		return nil, err
	}
	f.inc()
	return &cacheReader{
		r:   reader,
		cnt: &f.handleCounter,
	}, nil
}

func (f *cachedFile) Write(p []byte) (int, error) {
	return f.stream.Write(p)
}

func (f *cachedFile) Close() error {
	defer f.dec()
	return f.stream.Close()
}

type cacheReader struct {
	r   ReadAtCloser
	cnt *handleCounter
}

func (r *cacheReader) ReadAt(p []byte, off int64) (n int, err error) {
	return r.r.ReadAt(p, off)
}

func (r *cacheReader) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

func (r *cacheReader) Close() error {
	defer r.cnt.dec()
	return r.r.Close()
}

type handleCounter struct {
	cnt int64
	grp sync.WaitGroup
}

func (h *handleCounter) inc() {
	h.grp.Add(1)
	atomic.AddInt64(&h.cnt, 1)
}

func (h *handleCounter) dec() {
	atomic.AddInt64(&h.cnt, -1)
	h.grp.Done()
}

func (h *handleCounter) inUse() bool {
	return atomic.LoadInt64(&h.cnt) > 0
}

func (h *handleCounter) waitUntilFree() {
	h.grp.Wait()
}
