package watcher

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/devourer/server/internal/db"
	"github.com/devourer/server/internal/db/queries"
	"github.com/devourer/server/internal/metadata"
	"github.com/devourer/server/internal/scanner"
)

var targetExts = map[string]bool{
	".cbz": true, ".zip": true, ".cbr": true,
	".rar": true, ".pdf": true, ".epub": true,
	".7z": true, ".cb7": true,
	".mp3": true, ".m4a": true, ".m4b": true, ".ogg": true,
	".flac": true, ".aac": true, ".opus": true, ".wav": true,
}

func isTarget(path string) bool {
	return targetExts[strings.ToLower(filepath.Ext(path))]
}

const debounceDelay = 3 * time.Second

type Watcher struct {
	db          *sql.DB
	assetsPath  string
	pluginsPath string
	providers   map[string]*metadata.Provider
	fsw         *fsnotify.Watcher

	debounceMu sync.Mutex
	debounce   map[string]*time.Timer

	delMu    sync.Mutex
	delQueue []string
	delBusy  bool
}

func New(d *sql.DB, assetsPath, pluginsPath string, providers map[string]*metadata.Provider) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		db:          d,
		assetsPath:  assetsPath,
		pluginsPath: pluginsPath,
		providers:   providers,
		fsw:         fsw,
		debounce:    make(map[string]*time.Timer),
	}, nil
}

func (w *Watcher) Start() error {
	libs, err := queries.ListLibraries(w.db)
	if err != nil {
		return err
	}

	for _, lib := range libs {
		w.watchDir(lib.Path)
	}

	go w.loop()
	return nil
}

const maxWatchDepth = 3

func (w *Watcher) watchDir(root string) {
	w.watchDirDepth(root, 0)
}

func (w *Watcher) watchDirDepth(dir string, depth int) {
	if err := w.fsw.Add(dir); err != nil {
		log.Printf("[Watcher] Cannot watch %s: %v", dir, err)
		return
	}
	log.Printf("[Watcher] Watching: %s", dir)

	if depth >= maxWatchDepth {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == ".devourer" {
			continue
		}
		w.watchDirDepth(filepath.Join(dir, e.Name()), depth+1)
	}
}

func (w *Watcher) Stop() error {
	w.debounceMu.Lock()
	for _, t := range w.debounce {
		t.Stop()
	}
	w.debounceMu.Unlock()
	return w.fsw.Close()
}

func (w *Watcher) Restart() {
	w.fsw.Close()
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("[Watcher] Restart error: %v", err)
		return
	}
	w.fsw = fsw
	w.Start()
}

func (w *Watcher) debounceKeyFor(filePath string) string {
	lib := w.findLibrary(filePath)
	if lib == nil {
		return filePath
	}
	switch lib.Type {
	case "book":
		return filePath
	case "audiobook":
		return filepath.Dir(filePath)
	default: // manga
		rel, _ := filepath.Rel(lib.Path, filePath)
		parts := strings.SplitN(filepath.ToSlash(rel), "/", 2)
		if len(parts) > 0 {
			return filepath.Join(lib.Path, parts[0])
		}
		return filePath
	}
}

func (w *Watcher) loop() {
	for {
		select {
		case event, ok := <-w.fsw.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Create) {
				if filepath.Base(event.Name) != ".devourer" {
					if fi, err := os.Stat(event.Name); err == nil && fi.IsDir() {
						w.watchDir(event.Name)
					}
				}
			}

			switch {
			case event.Has(fsnotify.Create) || event.Has(fsnotify.Write):
				if isTarget(event.Name) {
					debounceKey := w.debounceKeyFor(event.Name)
					w.enqueueAdd(debounceKey, event.Name)
				}
			case event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename):
				if isTarget(event.Name) {
					w.enqueueDelete(event.Name)
				}
			}

		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			log.Printf("[Watcher] Error: %v", err)
		}
	}
}

func (w *Watcher) enqueueAdd(debounceKey, path string) {
	w.debounceMu.Lock()
	defer w.debounceMu.Unlock()

	if t, ok := w.debounce[debounceKey]; ok {
		t.Reset(debounceDelay)
		return
	}
	w.debounce[debounceKey] = time.AfterFunc(debounceDelay, func() {
		w.debounceMu.Lock()
		delete(w.debounce, debounceKey)
		w.debounceMu.Unlock()
		w.processAdd(path)
	})
}

func (w *Watcher) processAdd(path string) {
	lib := w.findLibrary(path)
	if lib == nil {
		return
	}

	cfg := &scanner.Config{DB: w.db, AssetsPath: w.assetsPath, PluginsPath: w.pluginsPath, Providers: w.providers}
	switch lib.Type {
	case "book":
		scanner.ProcessBook(cfg, lib, path)
	case "audiobook":
		scanner.ProcessAudiobook(cfg, lib, filepath.Dir(path))
	default:
		rel, _ := filepath.Rel(lib.Path, path)
		parts := strings.SplitN(filepath.ToSlash(rel), "/", 2)
		if len(parts) > 1 {
			scanner.ProcessManga(cfg, lib, parts[0])
		}
	}
}

func (w *Watcher) enqueueDelete(path string) {
	w.delMu.Lock()
	w.delQueue = append(w.delQueue, path)
	if !w.delBusy {
		w.delBusy = true
		go w.processDeletes()
	}
	w.delMu.Unlock()
}

func (w *Watcher) processDeletes() {
	for {
		w.delMu.Lock()
		if len(w.delQueue) == 0 {
			w.delBusy = false
			w.delMu.Unlock()
			return
		}
		path := w.delQueue[0]
		w.delQueue = w.delQueue[1:]
		w.delMu.Unlock()

		lib := w.findLibrary(path)
		if lib == nil {
			continue
		}
		switch lib.Type {
		case "book":
			scanner.DeleteBook(w.db, lib.Path, path)
		case "audiobook":
			scanner.DeleteAudiobook(w.db, lib.Path, path)
		default:
			scanner.DeleteManga(w.db, lib.Path, path)
		}
	}
}

func (w *Watcher) findLibrary(filePath string) *db.Library {
	libs, err := queries.ListLibraries(w.db)
	if err != nil {
		return nil
	}
	for _, lib := range libs {
		if strings.HasPrefix(filePath, lib.Path) {
			return lib
		}
	}
	return nil
}
