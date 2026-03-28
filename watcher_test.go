package devr

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatchRoots(t *testing.T) {
	base := "/tmp/project"

	got := watchRoots(base, []string{"cmd", "/var/tmp/abs"})

	assert.Equal(t, []string{"/tmp/project/cmd", "/var/tmp/abs"}, got)
}

func TestWatchRootsDefaultsToDot(t *testing.T) {
	got := watchRoots("/tmp/project", nil)
	assert.Equal(t, []string{"/tmp/project"}, got)
}

func TestShouldHandleEvent(t *testing.T) {
	tests := []struct {
		name string
		ev   fsnotify.Event
		want bool
	}{
		{name: "write matching extension", ev: fsnotify.Event{Name: "main.go", Op: fsnotify.Write}, want: true},
		{name: "create matching extension", ev: fsnotify.Event{Name: "main.go", Op: fsnotify.Create}, want: true},
		{name: "remove ignored", ev: fsnotify.Event{Name: "main.go", Op: fsnotify.Remove}, want: false},
		{name: "wrong extension ignored", ev: fsnotify.Event{Name: "main.txt", Op: fsnotify.Write}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldHandleEvent(tt.ev, []string{".go"}))
		})
	}
}

func TestShouldSkipDir(t *testing.T) {
	exclude := makeExcludeSet([]string{"vendor", "node_modules"})

	assert.True(t, shouldSkipDir(".git", exclude))
	assert.True(t, shouldSkipDir("vendor", exclude))
	assert.False(t, shouldSkipDir("cmd", exclude))
}

func TestAddDirectoriesSkipsExcludedAndHiddenDirs(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "cmd"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "vendor"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0755))

	w, err := fsnotify.NewWatcher()
	require.NoError(t, err)

	defer func() { _ = w.Close() }()

	require.NoError(t, addDirectories(w, root, []string{"vendor"}))

	got := map[string]bool{}
	for _, path := range w.WatchList() {
		got[path] = true
	}

	assert.True(t, got[root])
	assert.True(t, got[filepath.Join(root, "cmd")])
	assert.False(t, got[filepath.Join(root, "vendor")])
	assert.False(t, got[filepath.Join(root, ".git")])
}

func TestWatchDebouncesMatchingEvents(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "main.go")
	require.NoError(t, os.WriteFile(file, []byte("package main\n"), 0644))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls atomic.Int32

	done := make(chan struct{}, 1)

	go func() {
		err := Watch(ctx, root, WatchOptions{
			Extensions: []string{".go"},
			Debounce:   50 * time.Millisecond,
		}, func() {
			if calls.Add(1) == 1 {
				done <- struct{}{}
			}

			cancel()
		})
		assert.NoError(t, err)
	}()

	time.Sleep(150 * time.Millisecond)

	require.NoError(t, os.WriteFile(file, []byte("package main\n\n"), 0644))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, os.WriteFile(file, []byte("package main\n\nfunc main() {}\n"), 0644))

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("watch callback was not triggered")
	}

	assert.Equal(t, int32(1), calls.Load())
}

func TestMatchExt(t *testing.T) {
	tests := []struct {
		name string
		file string
		exts []string
		want bool
	}{
		{"match .go", "main.go", []string{".go"}, true},
		{"match .templ", "page.templ", []string{".go", ".templ"}, true},
		{"no match", "readme.md", []string{".go"}, false},
		{"empty exts", "main.go", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, matchExt(tt.file, tt.exts))
		})
	}
}

func TestMakeExcludeSet(t *testing.T) {
	set := makeExcludeSet([]string{"vendor", "tmp"})
	assert.Len(t, set, 2)
	_, ok := set["vendor"]
	assert.True(t, ok)
	_, ok = set["other"]
	assert.False(t, ok)
}

func TestWatchIgnoresExcludedDirectories(t *testing.T) {
	root := t.TempDir()
	excluded := filepath.Join(root, "vendor")
	require.NoError(t, os.Mkdir(excluded, 0755))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls atomic.Int32

	go func() {
		err := Watch(ctx, root, WatchOptions{
			Extensions: []string{".go"},
			Exclude:    []string{"vendor"},
			Debounce:   20 * time.Millisecond,
		}, func() {
			calls.Add(1)
		})
		assert.NoError(t, err)
	}()

	time.Sleep(150 * time.Millisecond)

	require.NoError(t, os.WriteFile(filepath.Join(excluded, "ignored.go"), []byte("package vendor\n"), 0644))
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, int32(0), calls.Load())
}
