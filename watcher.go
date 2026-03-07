package devr

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type WatchOptions struct {
	Dirs       []string
	Extensions []string
	Exclude    []string
	Debounce   time.Duration
}

func newWatchOptions(cfg ConfigWatch) WatchOptions {
	return WatchOptions(cfg)
}

func Watch(ctx context.Context, dir string, opts WatchOptions, onChange func()) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	defer func() { _ = w.Close() }()

	for _, root := range watchRoots(dir, opts.Dirs) {
		if err := addDirectories(w, root, opts.Exclude); err != nil {
			return err
		}
	}

	debounce := newStoppedTimer()
	defer debounce.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-w.Events:
			if !ok {
				return nil
			}

			if !shouldHandleEvent(event, opts.Extensions) {
				continue
			}

			resetTimer(debounce, opts.Debounce)
		case <-debounce.C:
			onChange()
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}

			Warn("watcher: %v", err)
		}
	}
}

func watchRoots(base string, dirs []string) []string {
	if len(dirs) == 0 {
		dirs = []string{"."}
	}

	roots := make([]string, 0, len(dirs))
	for _, d := range dirs {
		root := d
		if !filepath.IsAbs(root) {
			root = filepath.Join(base, root)
		}

		roots = append(roots, root)
	}

	return roots
}

func shouldHandleEvent(event fsnotify.Event, extensions []string) bool {
	if !matchExt(event.Name, extensions) {
		return false
	}

	return event.Op&(fsnotify.Write|fsnotify.Create) != 0
}

func matchExt(name string, extensions []string) bool {
	for _, ext := range extensions {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}

	return false
}

func newStoppedTimer() *time.Timer {
	timer := time.NewTimer(time.Hour)
	if !timer.Stop() {
		// Drain the initial tick if the timer fired before Stop won the race.
		select {
		case <-timer.C:
		default:
		}
	}

	return timer
}

func resetTimer(timer *time.Timer, d time.Duration) {
	if !timer.Stop() {
		// Reset requires an empty channel; drain any pending tick first.
		select {
		case <-timer.C:
		default:
		}
	}

	timer.Reset(d)
}

func addDirectories(w *fsnotify.Watcher, root string, exclude []string) error {
	excludeSet := makeExcludeSet(exclude)

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip inaccessible directories
		}

		if info.IsDir() {
			if shouldSkipDir(info.Name(), excludeSet) {
				return filepath.SkipDir
			}

			return w.Add(path)
		}

		return nil
	})
}

func makeExcludeSet(exclude []string) map[string]struct{} {
	excludeSet := make(map[string]struct{}, len(exclude))
	for _, e := range exclude {
		excludeSet[e] = struct{}{}
	}

	return excludeSet
}

func shouldSkipDir(name string, excludeSet map[string]struct{}) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}

	_, ok := excludeSet[name]

	return ok
}
