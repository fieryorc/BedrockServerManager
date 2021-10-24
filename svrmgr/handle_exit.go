package svrmgr

import (
	"context"
)

// exitHandler implements exit command.
type exitHandler struct{}

func initExitHandler(provider Provider) {
	provider.Register("exit", &exitHandler{})
}

func (h *exitHandler) Handle(ctx context.Context, provider Provider, cmd []string) error {
	provider.RunCommand(ctx, "stop")
	return ErrExit
}
