package svrmgr

import (
	"context"
	"fmt"
)

type stopHandler struct{}

func init() {
	Register("stop", &stopHandler{})
}

func (h *stopHandler) Handle(ctx context.Context, provider Provider, cmd []string) error {
	var err error
	proc := provider.GetServerProcess()
	if proc == nil {
		return fmt.Errorf("server not started")
	}
	if err = proc.Kill(); err != nil {
		return fmt.Errorf("unable to stop server")
	}
	provider.ResetServerProcess()
	return nil
}
