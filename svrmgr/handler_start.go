package svrmgr

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/golang/glog"
)

type startHandler struct {
	bedrockPath string
}

var bedrockServerExecutable = flag.String("bedrock_exe", "bedrock_server.exe", "Bedrock executable path. Defaults to current directory")

func init() {
	Register("start", &startHandler{
		bedrockPath: getBedrockServerPath(),
	})
}

var bedrockPath string

func getBedrockServerPath() string {
	if bedrockPath == "" {
		var err error
		exePath := *bedrockServerExecutable
		if !filepath.IsAbs(exePath) {
			exePath, err = exec.LookPath(*bedrockServerExecutable)
			if err != nil {
				panic(fmt.Sprintf("bedrock server not found. %v", err))
			}
			bedrockPath = exePath
		}
		glog.Infof("bedrockPath = %s", bedrockPath)
	}
	return bedrockPath

}

func (h *startHandler) Handle(ctx context.Context, provider Provider, command []string) error {
	cwd, _ := os.Getwd()
	cmd := exec.CommandContext(ctx, h.bedrockPath)
	cmd.Dir = cwd
	provider.SetServerProcess(cmd)

	return provider.GetServerProcess().Start(ctx, provider)
}
