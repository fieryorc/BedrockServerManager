package svrmgr

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

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

// List of all registered handlers
var handlers = map[string]Handler{}

// Register a handler for given command.
func Register(cmd string, handler Handler) {
	glog.Infof("Registering handler for %s", cmd)
	handlers[cmd] = handler
}

// ServerManager contains the main server manager logic
type ServerManager struct {
	// Maintains the bedrock server info.
	// Initialized and never nil.
	serverProcess *Process
	gw            *gitWrapper
}

// NewServerManager creates a new server manager
// Should be called only once.
func NewServerManager() *ServerManager {
	sm := &ServerManager{}
	sm.serverProcess = NewProcess(sm, nil)
	sm.loadPlugings()

	wsDir := *gitWorkspaceDir
	if wsDir == "" {
		wsDir = filepath.Dir(getBedrockServerPath())
	}
	sm.gw = newGitWrapper(wsDir)

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
func (sm *ServerManager) Process(args []string) error {
	reader := bufio.NewReader(os.Stdin)
	sm.printHelp()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Main interactive promt and user input handling.
	// TODO: Make it so that the server output automatically
	// reprints the prompt. Also, print server status in the prompt.
	glog.Infof("handlers = %v", handlers)
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

	h, ok := handlers[parts[0]]
	if !ok {
		sm.Log(fmt.Sprintf("invalid command '%s'\n", parts[0]))
		return nil
	}

	glog.Infof("Handler found, invoking")
	err = h.Handle(ctx, sm, parts)
	if err != nil {
		sm.Log(err.Error())
	}
	return nil
}
