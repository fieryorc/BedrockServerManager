package svrmgr

import (
	"context"
	"os"
)

type exitHandler struct{}

func init() {
	Register("exit", &exitHandler{})
}

func (h *exitHandler) Handle(ctx context.Context, provider Provider, cmd []string) error {
	provider.RunCommand(ctx, "stop")
	os.Exit(0)
	return nil
}
