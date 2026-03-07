package devr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func cmdInit(a *App) Command {
	return Command{
		Name: "init", Usage: "Scaffold a new Go project", Args: "<name>",
		Run: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: %s init <name>", a.Name)
			}

			name := args[0]

			if _, err := os.Stat(name); err == nil {
				return fmt.Errorf("'%s' already exists", name)
			}

			Info("Scaffolding %s...", name)

			if err := os.MkdirAll(filepath.Join(name, "cmd", name), 0755); err != nil {
				return err
			}

			mainGo := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
			if err := os.WriteFile(filepath.Join(name, "cmd", name, "main.go"), []byte(mainGo), 0644); err != nil {
				return err
			}

			gitignore := `/tmp/
*.exe
*.test
*.out
`
			if err := os.WriteFile(filepath.Join(name, ".gitignore"), []byte(gitignore), 0644); err != nil {
				return err
			}

			cmd := exec.Command("go", "mod", "init", name)

			cmd.Dir = name
			if err := cmd.Run(); err != nil {
				return err
			}

			cmd = exec.Command("git", "init", "-q")

			cmd.Dir = name
			if err := cmd.Run(); err != nil {
				Warn("git init failed: %v", err)
			}

			Info("Created %s/ with go.mod, cmd/%s/main.go, .gitignore", name, name)
			Info("  cd %s && %s app run", name, a.Name)

			return nil
		},
	}
}
