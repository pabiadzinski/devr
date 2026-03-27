package devr

import (
	"context"
	"fmt"
	"time"
)

func cmdApp(a *App) Command {
	return Command{
		Name: "app", Usage: "Manage application lifecycle",
		Sub: []Command{
			cmdRun(a),
			cmdWatch(a),
			cmdStop(a),
			cmdPs(a),
			cmdLogs(a),
			cmdAttach(a),
		},
	}
}

func cmdRun(a *App) Command {
	return Command{
		Name: "run", Usage: "Build & run Go app with log viewer", Args: "[pkg]",
		Flags: joinFlags(a.Cfg.BuildCLIFlags(), a.Cfg.RunCLIFlags()),
		Run: func(ctx context.Context, args []string) error {
			pkg, err := a.FindPkg(pkgArg(args))
			if err != nil {
				return err
			}

			a.Kill()

			pid, exitCh, err := a.BuildAndStart(pkg)
			if err != nil {
				return err
			}

			return a.runAppLogView("RUN", pid, exitCh, nil)
		},
	}
}

func cmdWatch(a *App) Command {
	var debounce string

	return Command{
		Name: "watch", Usage: "Rebuild & restart on .go file changes", Args: "[pkg]",
		Flags: joinFlags(
			a.Cfg.BuildCLIFlags(),
			a.Cfg.RunCLIFlags(),
			[]Flag{{Name: "debounce", Short: "d", Usage: "Debounce duration (e.g. 500ms, 1s)", Value: &debounce}},
		),
		Run: func(ctx context.Context, args []string) error {
			if debounce != "" {
				if d, err := time.ParseDuration(debounce); err == nil {
					a.Cfg.Watch.Debounce = d
				} else {
					return fmt.Errorf("invalid debounce: %s", debounce)
				}
			}

			pkg, err := a.FindPkg(pkgArg(args))
			if err != nil {
				return err
			}

			a.Kill()

			supervisor := a.newWatchSupervisor(ctx, pkg)

			pid, exitCh, err := supervisor.start()
			if err != nil {
				supervisor.reportBuildFailure(err)
			}

			Info("Watching for .go changes... (Ctrl+C to stop)")

			go func() {
				_ = Watch(ctx, a.WorkDir, newWatchOptions(a.Cfg.Watch), supervisor.rebuild)
			}()

			if pid > 0 {
				return a.runAppLogView("WATCH", pid, exitCh, supervisor.notifyIfCrash)
			}

			<-ctx.Done()

			return nil
		},
	}
}

func cmdStop(a *App) Command {
	return Command{
		Name: "stop", Usage: "Stop background process",
		Run: func(ctx context.Context, args []string) error {
			return a.Stop()
		},
	}
}

func cmdPs(a *App) Command {
	return Command{
		Name: "ps", Usage: "List all managed processes",
		Run: func(ctx context.Context, args []string) error {
			processes := a.listManagedProcesses()
			if len(processes) == 0 {
				Info("No running processes")
				return nil
			}

			fmt.Printf("%-20s %-8s %s\n", "NAME", "PID", "STATUS")

			for _, p := range processes {
				status := "stopped"
				if p.Running {
					status = "running"
				}

				fmt.Printf("%-20s %-8d %s\n", p.Name, p.PID, status)
			}

			return nil
		},
	}
}

func cmdLogs(a *App) Command {
	return Command{
		Name: "logs", Usage: "View last run's logs",
		Run: func(ctx context.Context, args []string) error {
			return RunLogView(LogViewOptions{
				LogPath:         a.LogFile(),
				Logs:            a.Cfg.Logs,
				HighlightFields: a.Cfg.Logs.HighlightFields,
			})
		},
	}
}

func (a *App) logViewTitle(title string) string {
	if label := a.Cfg.Build.Label(); label != "" {
		return title + " [" + label + "]"
	}

	return title
}

func (a *App) runAppLogView(title string, pid int, exitCh <-chan error, onExit func()) error {
	return RunLogView(LogViewOptions{
		LogPath:         a.LogFile(),
		PID:             pid,
		ExitCh:          exitCh,
		OnExit:          onExit,
		OnStop:          func() { _ = a.Stop() },
		Title:           a.logViewTitle(title),
		Logs:            a.Cfg.Logs,
		HighlightFields: a.Cfg.Logs.HighlightFields,
	})
}

func (a *App) reportBuildFailure(err error) {
	Error("%v", err)

	if a.Cfg.Notify {
		Notify(a.Name, "Build failed")
	}
}

type watchSupervisor struct {
	app       *App
	ctx       context.Context
	pkg       string
	cancelMon context.CancelFunc
}

func (a *App) newWatchSupervisor(ctx context.Context, pkg string) *watchSupervisor {
	return &watchSupervisor{
		app: a,
		ctx: ctx,
		pkg: pkg,
	}
}

func (w *watchSupervisor) start() (int, <-chan error, error) {
	pid, exitCh, err := w.app.BuildAndStart(w.pkg)
	if err != nil {
		return 0, nil, err
	}

	w.monitorExit(exitCh)

	return pid, exitCh, nil
}

func (w *watchSupervisor) rebuild() {
	Info("Rebuilding...")
	w.stopMonitoring()
	w.app.Kill()

	if _, exitCh, err := w.app.BuildAndStart(w.pkg); err != nil {
		w.reportBuildFailure(err)
		Info("Waiting for changes...")
	} else {
		w.monitorExit(exitCh)
	}
}

func (w *watchSupervisor) monitorExit(exitCh <-chan error) {
	w.stopMonitoring()

	monCtx, cancel := context.WithCancel(w.ctx)
	w.cancelMon = cancel

	go func() {
		select {
		case <-monCtx.Done():
		case <-exitCh:
			if !w.app.killing.Load() {
				Error("Process crashed")

				if w.app.Cfg.Notify {
					Notify(w.app.Name, "Process crashed")
				}

				Info("Waiting for changes...")
			}
		}
	}()
}

func (w *watchSupervisor) stopMonitoring() {
	if w.cancelMon != nil {
		w.cancelMon()
		w.cancelMon = nil
	}
}

func (w *watchSupervisor) notifyIfCrash() {
	if !w.app.killing.Load() && w.app.Cfg.Notify {
		Notify(w.app.Name, "Process crashed")
	}
}

func (w *watchSupervisor) reportBuildFailure(err error) {
	w.app.reportBuildFailure(err)
}

func cmdAttach(a *App) Command {
	return Command{
		Name: "attach", Usage: "Attach to running process logs",
		Run: func(ctx context.Context, args []string) error {
			pid, err := a.ReadPid()
			if err != nil {
				return err
			}

			err = RunLogView(LogViewOptions{
				LogPath:         a.LogFile(),
				PID:             pid,
				OnStop:          func() { _ = a.Stop() },
				Title:           "ATTACH",
				Logs:            a.Cfg.Logs,
				HighlightFields: a.Cfg.Logs.HighlightFields,
			})
			if err != nil {
				return err
			}

			Info("Detached. Process still running")
			Info("  %s app attach — reattach", a.Name)
			Info("  %s app stop   — stop process", a.Name)

			return nil
		},
	}
}
