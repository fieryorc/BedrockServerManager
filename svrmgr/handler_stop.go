package svrmgr

import (
	"context"
	"fmt"
)

// stopHandler - Stop running server.
type stopHandler struct{}

func initStopHandler(prov Provider) {
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
	return nil
}
