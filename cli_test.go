package devr

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppRun(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr string
	}{
		{name: "no args shows help", args: nil},
		{name: "help flag", args: []string{"help"}},
		{name: "command executes", args: []string{"greet"}, want: "hello"},
		{name: "command with args", args: []string{"greet", "world"}, want: "hello world"},
		{name: "unknown command", args: []string{"nope"}, wantErr: "unknown command: nope"},
		{name: "version", args: []string{"version"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string

			app := &CLI{Name: "test", Version: "1.0"}
			app.Add(Command{
				Name: "greet", Usage: "Say hello",
				Run: func(ctx context.Context, args []string) error {
					if len(args) > 0 {
						got = "hello " + args[0]
					} else {
						got = "hello"
					}

					return nil
				},
			})

			err := app.Run(tt.args)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)

			if tt.want != "" {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSubcommandRouting(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr string
	}{
		{name: "subcommand executes", args: []string{"group", "sub1"}, want: "sub1"},
		{name: "subcommand with args", args: []string{"group", "sub2", "arg"}, want: "sub2:arg"},
		{name: "group help on no args", args: []string{"group"}},
		{name: "group help explicit", args: []string{"group", "help"}},
		{name: "unknown subcommand", args: []string{"group", "nope"}, wantErr: "unknown command: group nope"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string

			app := &CLI{Name: "test"}
			app.Add(Command{
				Name: "group", Usage: "A group",
				Sub: []Command{
					{
						Name: "sub1", Usage: "First",
						Run: func(ctx context.Context, args []string) error {
							got = "sub1"
							return nil
						},
					},
					{
						Name: "sub2", Usage: "Second", Args: "[arg]",
						Run: func(ctx context.Context, args []string) error {
							if len(args) > 0 {
								got = "sub2:" + args[0]
							} else {
								got = "sub2"
							}

							return nil
						},
					},
				},
			})

			err := app.Run(tt.args)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)

			if tt.want != "" {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantStr  string
		wantBool bool
		wantRest []string
	}{
		{
			name:    "string flag long",
			args:    []string{"--name", "foo", "rest"},
			wantStr: "foo", wantRest: []string{"rest"},
		},
		{
			name:    "string flag short",
			args:    []string{"-n", "bar"},
			wantStr: "bar", wantRest: []string{},
		},
		{
			name:    "bool flag",
			args:    []string{"-v", "rest"},
			wantStr: "default", wantBool: true, wantRest: []string{"rest"},
		},
		{
			name:     "defaults when no flags",
			args:     []string{"rest"},
			wantStr:  "default",
			wantRest: []string{"rest"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				s string
				b bool
			)

			flags := []Flag{
				{Name: "name", Short: "n", Default: "default", Value: &s},
				{Name: "verbose", Short: "v", Bool: &b},
			}

			rest := parseFlags(flags, tt.args)

			assert.Equal(t, tt.wantStr, s)
			assert.Equal(t, tt.wantBool, b)
			assert.Equal(t, tt.wantRest, rest)
		})
	}
}

func TestSetupCalledBeforeCommand(t *testing.T) {
	var order []string

	app := &CLI{
		Name: "test",
		Setup: func() error {
			order = append(order, "setup")
			return nil
		},
	}
	app.Add(Command{
		Name: "cmd",
		Run: func(ctx context.Context, args []string) error {
			order = append(order, "cmd")
			return nil
		},
	})

	require.NoError(t, app.Run([]string{"cmd"}))
	assert.Equal(t, []string{"setup", "cmd"}, order)
}

func TestSetupError(t *testing.T) {
	app := &CLI{
		Name: "test",
		Setup: func() error {
			return fmt.Errorf("setup failed")
		},
	}
	app.Add(Command{
		Name: "cmd",
		Run: func(ctx context.Context, args []string) error {
			return nil
		},
	})

	err := app.Run([]string{"cmd"})
	require.EqualError(t, err, "setup failed")
}

func TestRootHelpAndVersionOutput(t *testing.T) {
	app := &CLI{
		Name:    "test",
		Version: "1.0",
		Flags:   []Flag{{Name: "dir", Short: "C", Usage: "Change directory"}},
	}
	app.Add(Command{Name: "greet", Usage: "Say hello", Args: "[name]"})

	tests := []struct {
		name    string
		args    []string
		stream  string
		want    []string
		wantErr string
	}{
		{
			name:   "no args prints help to stdout",
			args:   nil,
			stream: "stdout",
			want:   []string{"Usage: test [options] <command> [args]", "Commands:", "greet [name]", "help", "version"},
		},
		{
			name:   "help token prints help to stdout",
			args:   []string{"help"},
			stream: "stdout",
			want:   []string{"Run 'test help' for more information."},
		},
		{
			name:   "version prints version to stdout",
			args:   []string{"version"},
			stream: "stdout",
			want:   []string{"test 1.0"},
		},
		{
			name:    "unknown command prints help to stderr",
			args:    []string{"nope"},
			stream:  "stderr",
			want:    []string{"Usage: test [options] <command> [args]"},
			wantErr: "unknown command: nope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := captureOutput(t, func() {
				err := app.Run(tt.args)
				if tt.wantErr != "" {
					require.EqualError(t, err, tt.wantErr)
					return
				}

				require.NoError(t, err)
			})

			out := stdout
			if tt.stream == "stderr" {
				out = stderr
			}

			for _, want := range tt.want {
				assert.Contains(t, out, want)
			}
		})
	}
}

func TestSubcommandHelpOutput(t *testing.T) {
	app := &CLI{Name: "test"}
	app.Add(Command{
		Name:  "group",
		Usage: "A group",
		Sub: []Command{
			{
				Name:  "serve",
				Usage: "Serve requests",
				Args:  "[addr]",
				Flags: []Flag{{Name: "port", Short: "p", Usage: "Port to listen on", Default: "8080"}},
				Run: func(ctx context.Context, args []string) error {
					return nil
				},
			},
		},
	})

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{name: "explicit help subcommand", args: []string{"group", "help"}, want: []string{"Usage: test group <command> [args]", "serve [addr]"}},
		{name: "dash h subcommand help", args: []string{"group", "serve", "-h"}, want: []string{"Usage: test group serve [addr] [options]", "-p, --port", "default: 8080"}},
		{name: "double dash help subcommand help", args: []string{"group", "serve", "--help"}, want: []string{"Serve requests", "Options:"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := captureOutput(t, func() {
				require.NoError(t, app.Run(tt.args))
			})

			assert.Empty(t, strings.TrimSpace(stderr))

			for _, want := range tt.want {
				assert.Contains(t, stdout, want)
			}
		})
	}
}

func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()

	origStdout := os.Stdout
	origStderr := os.Stderr

	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)

	stderrR, stderrW, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = stdoutW
	os.Stderr = stderrW

	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	fn()

	require.NoError(t, stdoutW.Close())
	require.NoError(t, stderrW.Close())

	stdoutBytes, err := io.ReadAll(stdoutR)
	require.NoError(t, err)

	stderrBytes, err := io.ReadAll(stderrR)
	require.NoError(t, err)

	require.NoError(t, stdoutR.Close())
	require.NoError(t, stderrR.Close())

	return string(stdoutBytes), string(stderrBytes)
}
