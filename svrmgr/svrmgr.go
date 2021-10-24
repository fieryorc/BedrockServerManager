package svrmgr

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

//go:generate mockgen -package svrmgr -source=svrmgr.go -destination=svrmgr_mocks_test.go

// ErrExit is returned when the app exits.
var ErrExit = errors.New("exiting the session")

// aliases list
var aliases = map[string]string{
	"bs":        "backup save",
	"br":        "backup restore",
	"bl":        "backup list",
	"bp":        "backup period",
	"bd":        "backup delete",
	"workspace": "backup",
	"h":         "help",
	"e":         "exit",
	"q":         "exit",
	"quit":      "exit",
	"s":         "status",
	"$":         "shell",
	"wc":        "backup clean",
	"@":         "server",
}

var gitWorkspaceDir = flag.String("git_workspace", "", "git root directory for the world. If not specified, uses bedrock server directory")

// Handler for the plugins
type Handler interface {
	Handle(ctx context.Context, provider Provider, cmd []string) error
}

// ServerManager contains the main server manager logic
type ServerManager struct {
	// List of all registered handlers
	handlers map[string]Handler
	// Maintains the bedrock server info.
	// Initialized and never nil.
	serverProcess ServerProcess
	gw            GitWrapper
	stdin         io.Reader
	stdout        io.Writer
}

// NewServerManager creates a new server manager
// Should be called only once.
func NewServerManager() *ServerManager {
	sm := &ServerManager{}
	sm.serverProcess = NewProcess(sm, nil)
	sm.stdin = os.Stdin
	sm.stdout = os.Stdout
	sm.handlers = map[string]Handler{}

	sm.loadPlugings()
	wsDir := *gitWorkspaceDir
	if wsDir == "" {
		wsDir = filepath.Dir(getBedrockServerPath())
	}
	sm.gw = newGitWrapper(wsDir)

	return sm
}

//newServerManagerForTests create new servermanager for tests.
func newServerManagerForTests() *ServerManager {
	sm := &ServerManager{}
	sm.serverProcess = NewProcess(sm, nil)
	sm.handlers = map[string]Handler{}

	return sm
}

// loadPlugings loads all the plugins in the sytem.
// New plugin must be registered here.
func (sm *ServerManager) loadPlugings() {
	initExitHandler(sm)
	initHelpHandler(sm)
	initServerCmdHandler(sm)
	initShellCmdHandler(sm)
	initBackupHandler(sm)
	initStartHandler(sm)
	initStopHandler(sm)
	initStatusHandler(sm)
}

// printHelp - print interactive help message
func (sm *ServerManager) printHelp() {
	(&helpHandler{}).Handle(context.Background(), nil, nil)
}

// Process - sart the main loop
func (sm *ServerManager) Process(ctx context.Context, args []string) error {
	reader := bufio.NewReader(sm.stdin)
	sm.printHelp()

	// Main interactive promt and user input handling.
	// TODO: Make it so that the server output automatically reprints the prompt.
	// TODO: Print server status in the prompt.
	glog.Infof("handlers = %v", sm.handlers)
	for {
		sm.Printf("> ")
		cmd, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("unable to read error")
		}
		cmd = strings.Trim(cmd, " \r\n\t")
		if cmd == "" {
			continue
		}
		if err = sm.handleCommand(ctx, cmd); err != nil {
			if err == ErrExit {
				return nil
			}
			return err
		}
	}
}

// handleCommand handles a single command and dispatches to the plugin.
func (sm *ServerManager) handleCommand(ctx context.Context, cmd string) error {
	var err error
	// Expand aliases
	glog.Infof("handling command '%s'", cmd)
	parts := strings.Split(cmd, " ")
	al, ok := aliases[parts[0]]
	if ok {
		glog.Infof("alias found, '%s' = '%s'", parts[0], al)
		parts = append(strings.Split(al, " "), parts[1:]...)
		glog.Infof("expanded alias to '%s'", strings.Join(parts, " "))
	}

	h, ok := sm.handlers[parts[0]]
	if !ok {
		sm.Log(fmt.Sprintf("invalid command '%s'\n", parts[0]))
		return nil
	}

	glog.Infof("Handler found, invoking")
	err = h.Handle(ctx, sm, parts)
	if err != nil {
		sm.Log(err.Error())
	}

	if err == ErrExit {
		return err
	}

	return nil
}
