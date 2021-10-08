package svrmgr

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

// startHandler - start bedrock server.
type startHandler struct {
	bedrockPath string
}

var bedrockServerExecutable = flag.String("bedrock_exe", "bedrock_server.exe", "Bedrock executable path. Defaults to current directory")

func initStartHandler(prov Provider) {
	Register("start", &startHandler{
		bedrockPath: getBedrockServerPath(),
	})
}

var bedrockPath string

// getBedrockServerPath returns the executable path for the bedrock server.
// First it looks at the current directory and then looks at the PATH.
func getBedrockServerPath() string {
	if bedrockPath == "" {
		exePath := *bedrockServerExecutable
		if !filepath.IsAbs(exePath) {
			st, err := os.Stat(filepath.Join(".", *bedrockServerExecutable))
			if err == nil && !st.IsDir() {
				wd, _ := os.Getwd()
				bedrockPath = wd
			} else {
				exePath, err = exec.LookPath(*bedrockServerExecutable)
				if err != nil {
					panic(fmt.Sprintf("bedrock server not found. %v", err))
				}
				bedrockPath = exePath
			}
		}
		glog.Infof("bedrockPath = %s", bedrockPath)
	}
	return bedrockPath
}

// Handle - starts the server and waits for specific marker messages.
func (h *startHandler) Handle(ctx context.Context, provider Provider, command []string) error {
	cwd, _ := os.Getwd()
	cmd := exec.CommandContext(ctx, h.bedrockPath)
	cmd.Dir = cwd
	provider.SetServerProcess(cmd)

	ch := make(chan string)
	provider.GetServerProcess().StartReadOutput(ch)
	defer provider.GetServerProcess().EndReadOutput()

	err := provider.GetServerProcess().Start(ctx, provider)
	if err != nil {
		return err
	}

	count := 0
	for {
		l, ok := <-ch
		if !ok {
			return fmt.Errorf("failed to start the server")
		}
		// Second port message indicates server fully started.
		if strings.Contains(l, "[INFO] IPv6 supported, port:") {
			count += 1
			if count == 2 {
				glog.Infof("server started successfully")
				return nil
			}
		}
	}
}
