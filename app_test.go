package devr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppNew(t *testing.T) {
	dir := t.TempDir()

	a := NewApp("devr", dir)

	assert.Equal(t, "devr", a.Name)
	assert.Equal(t, filepath.Base(dir), a.Project)
	assert.Equal(t, dir, a.WorkDir)
}

func TestAppNewDefaultWorkDir(t *testing.T) {
	wd, _ := os.Getwd()

	a := NewApp("devr", "")

	assert.Equal(t, wd, a.WorkDir)
	assert.Equal(t, filepath.Base(wd), a.Project)
}

func TestAppFilePaths(t *testing.T) {
	a := NewApp("devr", "/tmp/myapp")
	tmp := os.TempDir()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"LogFile", a.LogFile(), filepath.Join(tmp, "devr-myapp.log")},
		{"BinFile", a.BinFile(), filepath.Join(tmp, "devr-myapp")},
		{"PidFile", a.PidFile(), filepath.Join(tmp, "devr-myapp.pid")},
		{"PidGlob", a.PidGlob(), filepath.Join(tmp, "devr-*.pid")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got)
		})
	}
}

func TestAppPrefix(t *testing.T) {
	tests := []struct {
		name    string
		appName string
		workDir string
		want    string
	}{
		{"standard", "devr", "/home/user/myapp", "devr-myapp"},
		{"custom name", "runner", "/home/user/svc", "runner-svc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewApp(tt.appName, tt.workDir)
			assert.Equal(t, tt.want, a.prefix())
		})
	}
}
