package devr

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadPidInvalidFileRemovesIt(t *testing.T) {
	app := NewApp("devr", t.TempDir())

	require.NoError(t, os.WriteFile(app.PidFile(), []byte("abc"), 0644))

	_, err := app.ReadPid()
	require.EqualError(t, err, "invalid pid file for "+app.Project)

	_, statErr := os.Stat(app.PidFile())
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestReadPidStoppedProcessRemovesIt(t *testing.T) {
	app := NewApp("devr", t.TempDir())

	require.NoError(t, os.WriteFile(app.PidFile(), []byte("999999"), 0644))

	_, err := app.ReadPid()
	require.EqualError(t, err, "process 999999 is not running")

	_, statErr := os.Stat(app.PidFile())
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestListManagedProcessesIncludesRunningAndStopped(t *testing.T) {
	app := NewApp("devrtest", t.TempDir())

	running := startSleepProcess(t)
	defer stopProcess(t, running)

	runningPath := filepath.Join(os.TempDir(), "devrtest-running.pid")
	stoppedPath := filepath.Join(os.TempDir(), "devrtest-stopped.pid")

	require.NoError(t, os.WriteFile(runningPath, []byte(strconv.Itoa(running.Process.Pid)), 0644))
	require.NoError(t, os.WriteFile(stoppedPath, []byte("999999"), 0644))
	t.Cleanup(func() { _ = os.Remove(runningPath) })

	processes := app.listManagedProcesses()
	require.Len(t, processes, 2)

	got := map[string]managedProcess{}
	for _, p := range processes {
		got[p.Name] = p
	}

	require.Contains(t, got, "running")
	require.Contains(t, got, "stopped")
	assert.True(t, got["running"].Running)
	assert.Equal(t, running.Process.Pid, got["running"].PID)
	assert.False(t, got["stopped"].Running)
	assert.Equal(t, 999999, got["stopped"].PID)

	_, statErr := os.Stat(stoppedPath)
	assert.ErrorIs(t, statErr, os.ErrNotExist)

	stopProcess(t, running)
}

func TestStopSignalsTrackedProcessAndRemovesPidFile(t *testing.T) {
	app := NewApp("devr", t.TempDir())
	cmd := startSleepProcess(t)

	require.NoError(t, os.WriteFile(app.PidFile(), []byte(strconv.Itoa(cmd.Process.Pid)), 0644))

	require.NoError(t, app.Stop())
	waitForCommandExit(t, cmd)

	_, statErr := os.Stat(app.PidFile())
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestRuntimeEnvLoadsConfiguredEnvFile(t *testing.T) {
	dir := t.TempDir()
	app := NewApp("devr", dir)
	app.Cfg.Run.EnvFile = ".env.test"

	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env.test"), []byte("FOO=bar\n"), 0644))

	env := app.runtimeEnv()

	assert.Contains(t, env, "FOO=bar")
}

func startSleepProcess(t *testing.T) *exec.Cmd {
	t.Helper()

	cmd := exec.Command("sleep", "30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	require.NoError(t, cmd.Start())

	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			_, _ = cmd.Process.Wait()
		}
	})

	return cmd
}

func stopProcess(t *testing.T, cmd *exec.Cmd) {
	t.Helper()

	if cmd.Process == nil || cmd.ProcessState != nil {
		return
	}

	require.NoError(t, cmd.Process.Signal(syscall.SIGTERM))
	waitForCommandExit(t, cmd)
}

func waitForCommandExit(t *testing.T, cmd *exec.Cmd) {
	t.Helper()

	done := make(chan error, 1)

	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(3 * time.Second):
		t.Fatal("process did not exit")
	case err := <-done:
		if err == nil {
			return
		}

		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
	}
}

func TestManagedProcessName(t *testing.T) {
	name := managedProcessName("devr", "/tmp/devr-myapp.pid")
	assert.Equal(t, "myapp", name)
}

func TestWritePidFile(t *testing.T) {
	app := NewApp("devr", t.TempDir())

	require.NoError(t, app.writePidFile(42))

	data, err := os.ReadFile(app.PidFile())
	require.NoError(t, err)
	assert.Equal(t, "42", strings.TrimSpace(string(data)))
}
