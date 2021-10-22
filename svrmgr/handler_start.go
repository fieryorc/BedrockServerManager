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

var serverOutputMarker = "[INFO] IPv6 supported, port:"

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
				bedrockPath = filepath.Join(wd, *bedrockServerExecutable)
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

	if provider.GetServerProcess().IsRunning() {
		return fmt.Errorf("server already running")
	}

	glog.Infof("initializing server")
	cwd, _ := os.Getwd()
	proc := provider.InitServer(ctx, h.bedrockPath, cwd, nil)

	ch := make(chan string)
	proc.StartReadOutput(ch)
	defer proc.EndReadOutput()

	glog.Infof("starting server")
	err := proc.Start(ctx, provider)
	if err != nil {
		return err
	}

	glog.Infof("waiting for server to start")
	count := 0
	for {
		l, ok := <-ch
		glog.Infof("server output: %v", l)
		if !ok {
			return fmt.Errorf("failed to start the server")
		}
		// Second port message indicates server fully started.
		if strings.Contains(l, serverOutputMarker) {
			count += 1
			if count == 2 {
				glog.Infof("server started successfully")
				return nil
			}
		}
	}
}
