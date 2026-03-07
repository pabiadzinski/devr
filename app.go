package devr

import (
	"os"
	"path/filepath"
	"sync/atomic"
)

type App struct {
	Name    string
	Project string
	WorkDir string
	Cfg     Config
	EnvFile string
	killing atomic.Bool
}

func NewApp(name, workDir string) *App {
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	return &App{
		Name:    name,
		Project: filepath.Base(workDir),
		WorkDir: workDir,
		Cfg:     LoadConfig(workDir),
	}
}

func (a *App) prefix() string {
	return a.Name + "-" + a.Project
}

func (a *App) LogFile() string {
	return filepath.Join(os.TempDir(), a.prefix()+".log")
}

func (a *App) BinFile() string {
	return filepath.Join(os.TempDir(), a.prefix())
}

func (a *App) PidFile() string {
	return filepath.Join(os.TempDir(), a.prefix()+".pid")
}

func (a *App) PidGlob() string {
	return filepath.Join(os.TempDir(), a.Name+"-*.pid")
}
