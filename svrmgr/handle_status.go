package svrmgr

import (
	"context"
)

// statusHandler implements status command.
type statusHandler struct{}

func initStatusHandler(provider Provider) {
	Register("status", &statusHandler{})
}

func (h *statusHandler) Handle(ctx context.Context, provider Provider, cmd []string) error {
	if provider.GetServerProcess().IsRunning() {
		provider.Log("server is running")
	} else {
		provider.Log("server is not running")
	}
	return nil
}
