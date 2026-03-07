package devr

import (
	"context"
	"os"
	"os/exec"
)

func cmdTest(a *App) Command {
	return Command{
		Name: "test", Usage: "Run tests and benchmarks",
		Sub: []Command{cmdTestRun(), cmdBench(), cmdCover(a)},
	}
}

func cmdTestRun() Command {
	var (
		format  string
		short   bool
		run     string
		timeout string
	)

	return Command{
		Name: "run", Usage: "Run tests", Args: "[pkg]",
		Flags: []Flag{
			{Name: "format", Short: "f", Usage: "Output: dots, short, testname, verbose", Default: fmtTestname, Value: &format},
			{Name: "short", Short: "s", Usage: "Short mode", Bool: &short},
			{Name: "run", Short: "r", Usage: "Run only matching tests", Value: &run},
			{Name: "timeout", Short: "t", Usage: "Timeout (e.g. 30s, 5m)", Value: &timeout},
		},
		Run: func(ctx context.Context, args []string) error {
			pkg := pkgArg(args)
			if pkg == "" {
				pkg = "./..."
			}

			goArgs := []string{"test", "-json", "-count=1"}
			if format == fmtVerbose {
				goArgs = append(goArgs, "-v")
			}

			if short {
				goArgs = append(goArgs, "-short")
			}

			if run != "" {
				goArgs = append(goArgs, "-run", run)
			}

			if timeout != "" {
				goArgs = append(goArgs, "-timeout", timeout)
			}

			goArgs = append(goArgs, pkg)

			cmd := exec.Command("go", goArgs...)
			cmd.Stderr = os.Stderr

			stdout, err := cmd.StdoutPipe()
			if err != nil {
				return err
			}

			if err := cmd.Start(); err != nil {
				return err
			}

			tf := newTestFormatter(os.Stdout, format)
			tf.Process(stdout)
			tf.Summary()

			return cmd.Wait()
		},
	}
}

func cmdBench() Command {
	return Command{
		Name: "bench", Usage: "Run benchmarks", Args: "[pkg]",
		Run: func(ctx context.Context, args []string) error {
			pkg := pkgArg(args)
			if pkg == "" {
				pkg = "./..."
			}

			cmd := exec.Command("go", "test", "-bench=.", "-benchmem", "-run=^$", pkg)

			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			return cmd.Run()
		},
	}
}

func cmdCover(a *App) Command {
	return Command{
		Name: "cover", Usage: "Run tests with coverage", Args: "[pkg]",
		Run: func(ctx context.Context, args []string) error {
			pkg := pkgArg(args)
			if pkg == "" {
				pkg = "./..."
			}

			profile := a.Cfg.Test.CoverProfile
			cmd := exec.Command("go", "test",
				"-coverprofile="+profile, "-covermode=atomic", "-count=1", pkg)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				return err
			}

			Info("Coverage saved to %s", profile)

			htmlCmd := exec.Command("go", "tool", "cover", "-html="+profile)

			return htmlCmd.Run()
		},
	}
}
