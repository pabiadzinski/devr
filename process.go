package devr

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type managedProcess struct {
	Name    string
	PID     int
	Running bool
}

func (a *App) FindPkg(pkg string) (string, error) {
	if pkg != "" {
		return pkg, nil
	}

	matches, _ := filepath.Glob(filepath.Join(a.WorkDir, a.Cfg.Build.CmdPattern))
	if len(matches) > 0 {
		rel, _ := filepath.Rel(a.WorkDir, filepath.Dir(matches[0]))
		return "./" + rel, nil
	}

	if _, err := os.Stat(filepath.Join(a.WorkDir, "main.go")); err == nil {
		return ".", nil
	}

	return "", fmt.Errorf("could not find main.go, specify package path")
}

func (a *App) Build(pkg string) error {
	if label := a.Cfg.Build.Label(); label != "" {
		Info("Building %s [%s]...", pkg, label)
	} else {
		Info("Building %s...", pkg)
	}

	goFlags := a.Cfg.Build.GoFlags()
	args := make([]string, 0, 2+len(goFlags)+3)
	args = append(args, "build")
	args = append(args, goFlags...)
	args = append(args, "-o", a.BinFile(), pkg)
	cmd := exec.Command("go", args...)
	cmd.Dir = a.WorkDir

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
	}

	return nil
}

func (a *App) Start() (int, <-chan error, error) {
	a.killing.Store(false)

	logFile, err := os.Create(a.LogFile())
	if err != nil {
		return 0, nil, err
	}

	cmd := exec.Command(a.BinFile())
	cmd.Dir = a.WorkDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = a.runtimeEnv()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return 0, nil, err
	}

	_ = logFile.Close()

	pid := cmd.Process.Pid
	_ = a.writePidFile(pid)

	exitCh := make(chan error, 1)

	go func() {
		exitCh <- cmd.Wait()
	}()

	return pid, exitCh, nil
}

func (a *App) Stop() error {
	pid, err := a.signalTrackedProcess(syscall.SIGTERM, false)
	if err != nil {
		return err
	}

	Info("Stopped PID %d", pid)

	return nil
}

func (a *App) Kill() {
	a.killing.Store(true)

	pid, ok := a.currentPID()
	if !ok {
		return
	}

	if err := signalPID(pid, syscall.SIGTERM); err != nil {
		_ = os.Remove(a.PidFile())
		return
	}

	Dbg("Sending SIGTERM to PID %d", pid)

	for i := 0; i < 20; i++ {
		if !isProcessRunning(pid) {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	_ = os.Remove(a.PidFile())
}

func (a *App) ReadPid() (int, error) {
	return a.readTrackedPID()
}

func (a *App) BuildAndStart(pkg string) (int, <-chan error, error) {
	if err := a.Build(pkg); err != nil {
		return 0, nil, fmt.Errorf("build failed: %w", err)
	}

	pid, exitCh, err := a.Start()
	if err != nil {
		return 0, nil, fmt.Errorf("start failed: %w", err)
	}

	Info("Started PID %d, logging to %s", pid, a.LogFile())

	return pid, exitCh, nil
}

func (a *App) runtimeEnv() []string {
	env := os.Environ()

	if a.Cfg.Run.NoEnv {
		return env
	}

	envFile := a.Cfg.Run.EnvFile
	path := filepath.Join(a.WorkDir, envFile)

	if extra, err := loadEnvFile(path); err == nil {
		env = append(env, extra...)
	}

	return env
}

func (a *App) writePidFile(pid int) error {
	return os.WriteFile(a.PidFile(), []byte(strconv.Itoa(pid)), 0644)
}

func (a *App) readTrackedPID() (int, error) {
	pid, state, err := readManagedProcess(a.PidFile(), a.Project)
	if err != nil {
		return 0, err
	}

	if !state.Running {
		return 0, fmt.Errorf("process %d is not running", pid)
	}

	return pid, nil
}

func (a *App) currentPID() (int, bool) {
	pid, _, err := readManagedProcess(a.PidFile(), a.Project)
	return pid, err == nil
}

func (a *App) signalTrackedProcess(sig syscall.Signal, wait bool) (int, error) {
	pid, err := a.readTrackedPID()
	if err != nil {
		return 0, err
	}

	if err := signalPID(pid, sig); err != nil {
		_ = os.Remove(a.PidFile())
		return 0, err
	}

	if wait {
		waitForExit(pid, 20, 100*time.Millisecond)
	}

	_ = os.Remove(a.PidFile())

	return pid, nil
}

func (a *App) listManagedProcesses() []managedProcess {
	matches, _ := filepath.Glob(a.PidGlob())
	processes := make([]managedProcess, 0, len(matches))

	for _, pidPath := range matches {
		pid, state, err := readManagedProcess(pidPath, "")
		if err != nil {
			continue
		}

		processes = append(processes, managedProcess{
			Name:    managedProcessName(a.Name, pidPath),
			PID:     pid,
			Running: state.Running,
		})
	}

	return processes
}

type trackedPIDState struct {
	Running bool
}

func readManagedProcess(pidPath, project string) (int, trackedPIDState, error) {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		if project != "" {
			return 0, trackedPIDState{}, fmt.Errorf("no running process for %s", project)
		}

		return 0, trackedPIDState{}, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		_ = os.Remove(pidPath)

		if project != "" {
			return 0, trackedPIDState{}, fmt.Errorf("invalid pid file for %s", project)
		}

		return 0, trackedPIDState{}, err
	}

	running := isProcessRunning(pid)
	if !running {
		_ = os.Remove(pidPath)

		if project != "" {
			return 0, trackedPIDState{}, fmt.Errorf("process %d is not running", pid)
		}
	}

	return pid, trackedPIDState{Running: running}, nil
}

func managedProcessName(appName, pidPath string) string {
	base := filepath.Base(pidPath)
	name := strings.TrimPrefix(base, appName+"-")

	return strings.TrimSuffix(name, ".pid")
}

func signalPID(pid int, sig syscall.Signal) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	return p.Signal(sig)
}

func isProcessRunning(pid int) bool {
	return signalPID(pid, syscall.Signal(0)) == nil
}

func waitForExit(pid, attempts int, delay time.Duration) {
	for i := 0; i < attempts; i++ {
		if !isProcessRunning(pid) {
			return
		}

		time.Sleep(delay)
	}
}
