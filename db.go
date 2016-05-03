package fstack

import (
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/reddec/file-stack"
)

// Database of file stacks
type Database struct {
	io.Closer
	fileLock  sync.RWMutex
	files     map[string]*fstack.Stack
	collector *time.Ticker
	keepAlive time.Duration
	rootDir   string
}

//Find a stack or create new
func (db *Database) Find(key string, create bool) (*fstack.Stack, error) {
	var err error
	// Double check
	db.fileLock.RLock()
	fs, ok := db.files[key]
	db.fileLock.RUnlock()
	if !ok {
		fileName := filepath.Join(db.rootDir, url.QueryEscape(key))
		db.fileLock.Lock()
		defer db.fileLock.Unlock()
		if _, err := os.Stat(fileName); os.IsNotExist(err) && !create {
			return nil, nil
		}
		if fs, ok = db.files[key]; !ok {
			log.Println("New stack allocated at", fileName)
			fs, err = fstack.OpenStack(fileName)
		}
		if err == nil {
			db.files[key] = fs
		}
		if err != nil {
			return nil, err
		}
	}
	return fs, nil
}

// Get stack or create new. Panics on errors
func (db *Database) Get(key string) *fstack.Stack {
	s, err := db.Find(key, true)
	if err != nil {
		panic(err)
	}
	return s
}

// Close all allocated stacks and stops stack collector.
// Never use database again after close
func (db *Database) Close() error {
	db.fileLock.Lock()
	defer db.fileLock.Unlock()
	for _, s := range db.files {
		s.Close()
	}
	db.collector.Stop()
	return nil
}

func (db *Database) cleanup() {
	for _ = range db.collector.C {
		func() {
			db.fileLock.RLock()
			defer db.fileLock.RUnlock()
			n := time.Now()
			for _, s := range db.files {
				if n.Sub(s.LastAccess()) > db.keepAlive {
					s.Close()
				}
			}
		}()
	}
}

// Remove stack from database and file system
func (db *Database) Remove(key string) error {
	db.fileLock.RLock()
	fs, ok := db.files[key]
	if !ok {
		db.fileLock.RUnlock()
		return nil
	}
	db.fileLock.RUnlock()
	db.fileLock.Lock()
	defer db.fileLock.Unlock()
	fs, ok = db.files[key]
	if ok {
		fileName := filepath.Join(db.rootDir, url.QueryEscape(key))
		fs.Close()
		delete(db.files, key)
		return os.Remove(fileName)
	}
	return nil
}

// Clean and remove all stacks in database from filesystem
func (db *Database) Clean() error {
	db.fileLock.Lock()
	defer db.fileLock.Unlock()
	var err error
	for key, s := range db.files {
		s.Close()
		fileName := filepath.Join(db.rootDir, url.QueryEscape(key))
		e := os.Remove(fileName)
		if err == nil {
			err = e
		}
	}
	db.files = nil
	return err
}

// Scan root dir for allocated stacks
// Warning! All files in root dir will be interpreted as stacks
func (db *Database) Scan() error {
	db.fileLock.Lock()
	defer db.fileLock.Unlock()
	return filepath.Walk(db.rootDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		key, err := url.QueryUnescape(info.Name())
		if err != nil {
			return err
		}
		if _, ok := db.files[key]; !ok {
			fileName := filepath.Join(db.rootDir, info.Name())
			stack, err := fstack.OpenStack(fileName)
			if err != nil {
				return err
			}
			log.Println("Found stack allocated at", fileName, "mapped to", key, "with", stack.Depth(), "segments")
			db.files[key] = stack
		}
		return nil
	})
}

// Names of known stacks in the database
func (db *Database) Names() []string {
	db.fileLock.RLock()
	defer db.fileLock.RUnlock()
	names := []string{}
	for name := range db.files {
		names = append(names, name)
	}
	return names
}

//NewDatabase - create new database and start stack collector (closes outaded stack)
func NewDatabase(rootDir string, keepAlive time.Duration) (*Database, error) {
	err := os.MkdirAll(rootDir, 0755)
	if err != nil {
		return nil, err
	}
	db := &Database{
		files:     make(map[string]*fstack.Stack),
		rootDir:   rootDir,
		keepAlive: keepAlive,
		collector: time.NewTicker(keepAlive / 3),
	}

	go db.cleanup()
	return db, nil
}

// TODO: Sub sections?
// TODO: HTT API names
