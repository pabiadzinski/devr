package devr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPkgArg(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"./cmd/app"}, "./cmd/app"},
		{[]string{"./cmd/app", "extra"}, "./cmd/app"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, pkgArg(tt.args))
	}
}

func TestRegister(t *testing.T) {
	cli := &CLI{Name: "devr"}
	app := NewApp("devr", ".")
	Register(cli, app)

	names := make(map[string]bool)
	for _, cmd := range cli.commands {
		names[cmd.Name] = true
	}

	assert.True(t, names["app"])
	assert.True(t, names["test"])
	assert.True(t, names["init"])
	assert.True(t, names["completion"])
}
