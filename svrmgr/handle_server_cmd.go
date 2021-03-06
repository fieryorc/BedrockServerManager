package svrmgr

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"
)

// serverCmdHandler implements server commands.
// All commands will be sent to bedrock server directly.
type serverCmdHandler struct{}

func initServerCmdHandler(provider Provider) {
	provider.Register("server", &serverCmdHandler{})
}

func (h *serverCmdHandler) Handle(ctx context.Context, provider Provider, args []string) error {
	if len(args) < 2 {
		return nil
	}
	if !provider.GetServerProcess().IsRunning() {
		return fmt.Errorf("cannot send command. server is not running")
	}
	glog.Infof("sending command to bedrock server: %s %s", args[1], strings.Join(args[2:], " "))
	return provider.GetServerProcess().SendInput(strings.Join(args[1:], " "))
}
