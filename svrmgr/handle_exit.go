package svrmgr

import (
	"context"
	"os"
)

// exitHandler implements exit command.
type exitHandler struct{}

func initExitHandler(provider Provider) {
	Register("exit", &exitHandler{})
}

func (h *exitHandler) Handle(ctx context.Context, provider Provider, cmd []string) error {
	provider.RunCommand(ctx, "stop")
	os.Exit(0)
	return nil
}
