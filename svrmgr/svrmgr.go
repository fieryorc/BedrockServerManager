package svrmgr

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
)

var aliases = map[string]string{
	"bs": "backup save",
	"br": "backup restore",
	"bl": "backup list",
	"bp": "backup period",
	"h":  "help",
	"e":  "exit",
	"s":  "status",
	"$":  "shell",
	"@":  "server",
}

// Plugin interface
type Handler interface {
	Handle(ctx context.Context, provider Provider, cmd []string) error
}

var handlers = map[string]Handler{}

func Register(cmd string, handler Handler) {
	glog.Infof("Registering handler for %s", cmd)
	handlers[cmd] = handler
}

type ServerManager struct {
	serverProcess *Process
}

func NewServerManager() *ServerManager {
	sm := &ServerManager{}
	sm.serverProcess = NewProcess(sm, nil)
	sm.loadPlugings()
	return sm
}

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

func (sm *ServerManager) printHelp() {
	(&helpHandler{}).Handle(context.Background(), nil, nil)
}

func (sm *ServerManager) Process(args []string) error {
	reader := bufio.NewReader(os.Stdin)
	sm.printHelp()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
