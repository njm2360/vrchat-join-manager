package core

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"maps"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	stateSaveInterval = 60 * time.Second
	pollInterval      = 1 * time.Second
	scanInterval      = 10 * time.Second
	idleTimeout       = 1800 * time.Second
	logPattern        = "output_log_*.txt"
)

type WatcherState struct {
	path string
}

func newWatcherState(path string) *WatcherState {
	return &WatcherState{path: path}
}

func (ws *WatcherState) load() map[string]int64 {
	f, err := os.Open(ws.path)
	if err != nil {
		return make(map[string]int64)
	}
	defer f.Close()
	var m map[string]int64
	if err := json.NewDecoder(f).Decode(&m); err != nil {
		return make(map[string]int64)
	}
	log.Printf("State loaded: %s (%d entries)", ws.path, len(m))
	return m
}

func (ws *WatcherState) save(offsets map[string]int64, verbose bool) {
	tmp := ws.path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		log.Printf("state save create: %v", err)
		return
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(offsets); encErr != nil {
		f.Close()
		log.Printf("state save encode: %v", encErr)
		return
	}
	f.Close()
	if err := os.Rename(tmp, ws.path); err != nil {
		log.Printf("state save rename: %v", err)
		return
	}
	if verbose {
		log.Printf("State saved: %s", ws.path)
	}
}

type LineHandler func(path, line string)

type LogWatcher struct {
	logDir      string
	newHandler  func(path string) LineHandler
	state       *WatcherState
	readFromEnd bool

	mu         sync.Mutex
	offsets    map[string]int64
	knownFiles map[string]bool
	cancelFns  map[string]context.CancelFunc
	handlers   map[string]LineHandler

	fileWg sync.WaitGroup
}

func NewLogWatcher(logDir string, newHandler func(path string) LineHandler, stateFile string, readFromEnd bool) *LogWatcher {
	if stateFile == "" {
		exePath, _ := os.Executable()
		stateFile = filepath.Join(filepath.Dir(exePath), "state.json")
	}
	ws := newWatcherState(stateFile)
	offsets := ws.load()

	// Remove stale entries
	for path := range offsets {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Printf("Removed stale state entry: %s", path)
			delete(offsets, path)
		}
	}

	return &LogWatcher{
		logDir:      logDir,
		newHandler:  newHandler,
		state:       ws,
		readFromEnd: readFromEnd,
		offsets:     offsets,
		knownFiles:  make(map[string]bool),
		cancelFns:   make(map[string]context.CancelFunc),
		handlers:    make(map[string]LineHandler),
	}
}

func (w *LogWatcher) Run(ctx context.Context) error {
	var wg sync.WaitGroup

	wg.Go(func() { w.scanLoop(ctx) })
	wg.Go(func() { w.stateSaveLoop(ctx) })

	<-ctx.Done()

	// cancel all active file watchers
	w.mu.Lock()
	for _, cancel := range w.cancelFns {
		cancel()
	}
	w.mu.Unlock()

	wg.Wait()
	w.fileWg.Wait()

	// final state save
	w.mu.Lock()
	offsets := maps.Clone(w.offsets)
	w.mu.Unlock()
	w.state.save(offsets, true)

	return nil
}

func (w *LogWatcher) startWatchFile(ctx context.Context, path string) {
	w.mu.Lock()
	if _, already := w.cancelFns[path]; already {
		w.mu.Unlock()
		return
	}
	fileCtx, cancel := context.WithCancel(ctx)
	w.knownFiles[path] = true
	w.cancelFns[path] = cancel
	handler := w.newHandler(path)
	w.handlers[path] = handler
	w.mu.Unlock()

	w.fileWg.Go(func() {
		defer cancel()
		defer func() {
			w.mu.Lock()
			delete(w.cancelFns, path)
			delete(w.handlers, path)
			w.mu.Unlock()
		}()
		w.watchFile(fileCtx, path, handler)
	})
}

func (w *LogWatcher) watchFile(ctx context.Context, path string, handler LineHandler) {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("watchFile open %s: %v", path, err)
		return
	}
	defer f.Close()

	w.mu.Lock()
	offset, hasOffset := w.offsets[path]
	w.mu.Unlock()

	if hasOffset {
		if offset > 0 {
			if _, err := f.Seek(offset, io.SeekStart); err != nil {
				log.Printf("watchFile seek %s: %v", path, err)
				return
			}
		}
	} else if w.readFromEnd {
		endPos, err := f.Seek(0, io.SeekEnd)
		if err != nil {
			log.Printf("watchFile seek-end %s: %v", path, err)
			return
		}
		w.mu.Lock()
		w.offsets[path] = endPos
		w.mu.Unlock()
	}

	jitter := rand.N(pollInterval)
	select {
	case <-ctx.Done():
		return
	case <-time.After(jitter):
	}

	lastActive := time.Now()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		lines, newOffset, err := readLines(f)
		if err != nil {
			log.Printf("watchFile read %s: %v", path, err)
			return
		}

		if len(lines) > 0 {
			lastActive = time.Now()
			w.mu.Lock()
			w.offsets[path] = newOffset
			w.mu.Unlock()
			for _, line := range lines {
				handler(path, line)
			}
		} else if time.Since(lastActive) >= idleTimeout {
			log.Printf("File is stale. Remove from monitoring task: %s", path)
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *LogWatcher) scanLoop(ctx context.Context) {
	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	w.doScan(ctx, true)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.doScan(ctx, false)
		}
	}
}

func (w *LogWatcher) doScan(ctx context.Context, firstScan bool) {
	matches, err := filepath.Glob(filepath.Join(w.logDir, logPattern))
	if err != nil {
		log.Printf("scan glob: %v", err)
		return
	}

	w.mu.Lock()
	knownSnap := maps.Clone(w.knownFiles)
	offsetSnap := maps.Clone(w.offsets)
	activeSnap := make(map[string]bool, len(w.cancelFns))
	for k := range w.cancelFns {
		activeSnap[k] = true
	}
	w.mu.Unlock()

	for _, path := range matches {
		if !knownSnap[path] {
			if !firstScan {
				if _, exists := offsetSnap[path]; !exists {
					w.mu.Lock()
					w.offsets[path] = 0
					w.mu.Unlock()
					offsetSnap[path] = 0
				}
			}
			offset := offsetSnap[path]
			if offset > 0 {
				info, statErr := os.Stat(path)
				if statErr != nil {
					continue
				}
				if info.Size() <= offset {
					w.mu.Lock()
					w.knownFiles[path] = true
					w.mu.Unlock()
					continue
				}
			}
			w.startWatchFile(ctx, path)
			log.Printf("Monitoring start: %s", path)
			continue
		}

		if !activeSnap[path] {
			info, statErr := os.Stat(path)
			if statErr != nil {
				w.mu.Lock()
				delete(w.knownFiles, path)
				w.mu.Unlock()
				continue
			}
			if info.Size() > offsetSnap[path] {
				w.startWatchFile(ctx, path)
				log.Printf("Monitoring resume: %s", path)
			}
		}
	}
}

func (w *LogWatcher) stateSaveLoop(ctx context.Context) {
	ticker := time.NewTicker(stateSaveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.mu.Lock()
			offsets := maps.Clone(w.offsets)
			w.mu.Unlock()
			w.state.save(offsets, false)
		}
	}
}

func readLines(f *os.File) (lines []string, newOffset int64, err error) {
	startPos, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, 0, err
	}

	reader := bufio.NewReader(f)
	pos := startPos

	for {
		raw, readErr := reader.ReadString('\n')
		if strings.HasSuffix(raw, "\n") {
			pos += int64(len(raw))
			lines = append(lines, strings.TrimRight(raw, "\r\n"))
		}
		if readErr != nil {
			break
		}
	}

	if _, err := f.Seek(pos, io.SeekStart); err != nil {
		return lines, pos, err
	}
	return lines, pos, nil
}
