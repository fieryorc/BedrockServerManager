package svrmgr

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/golang/glog"
)

type shellCmdHandler struct{}

func initShellCmdHandler(prov Provider) {
	Register("shell", &shellCmdHandler{})
}

func (h *shellCmdHandler) Handle(ctx context.Context, provider Provider, args []string) error {
	if len(args) < 2 {
		return nil
	}
	glog.Infof("running %s %s", args[1], strings.Join(args[2:], " "))
	cmd := exec.CommandContext(ctx, args[1], args[2:]...)
	out, err := cmd.CombinedOutput()
	provider.Log(fmt.Sprintf(" %s", string(out)))
	if err != nil {
		provider.Log("command failed")
		return err
	}

	return nil
}
