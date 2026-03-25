package devr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testCLI() *CLI {
	a := &CLI{Name: "testcli"}
	app := NewApp("testcli", ".")
	Register(a, app)

	return a
}

func TestCompletionFish(t *testing.T) {
	cli := testCLI()
	out := cli.completionFish()

	assert.Contains(t, out, "complete -c testcli")
	assert.Contains(t, out, "__fish_use_subcommand")
	assert.Contains(t, out, "fish bash zsh")
}

func TestCompletionBash(t *testing.T) {
	cli := testCLI()
	out := cli.completionBash()

	assert.Contains(t, out, "_testcli()")
	assert.Contains(t, out, "compgen")
	assert.Contains(t, out, "complete -F _testcli testcli")
}

func TestCompletionZsh(t *testing.T) {
	cli := testCLI()
	out := cli.completionZsh()

	assert.Contains(t, out, "#compdef testcli")
	assert.Contains(t, out, "_testcli()")
	assert.Contains(t, out, "_describe")
	assert.Contains(t, out, "fish bash zsh")
}
