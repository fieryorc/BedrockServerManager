package svrmgr

import (
	"context"
	"fmt"
)

// statusHandler implements status command.
type statusHandler struct{}

func initStatusHandler(provider Provider) {
	provider.Register("status", &statusHandler{})
}

func (h *statusHandler) Handle(ctx context.Context, provider Provider, cmd []string) error {
	serverState := "not running"
	if provider.GetServerProcess().IsRunning() {
		serverState = "running"
	}

	isClean, err := provider.GitWrapper().IsDirClean(ctx)
	if err != nil {
		return err
	}

	wsState := "clean"
	if !isClean {
		wsState = "dirty"
	}

	bhI, _ := provider.GetHandler("backup")
	bh := bhI.(*backupHandler)
	backupStatus := bh.Status(ctx, provider)

	provider.Log(fmt.Sprintf(`server is %s, workspace is %s, %s`, serverState, wsState, backupStatus))
	return nil
}
