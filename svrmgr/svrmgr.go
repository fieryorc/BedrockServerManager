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
	"bf": "backup save",
	"br": "backup restore",
	"bl": "backup list",
	"bp": "backup period",
	"e":  "exit",
}

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
	return &ServerManager{}
}

func (sm *ServerManager) Process(args []string) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(`Welcome to Minecraft Bedrock Server Manager for Windows.

Syntax:
	@ COMMAND
		Send commands minecraft server directly.
	$ COMMAND
		Execute the shell command directly and print output.
	start
		Start the bedrock server
	stop
		Stop the bedrock server
	exit
		Exit the server manager shell. If server is running, will be stopped.
		alias: e
	backup save [Optional backup message]
		Take a backup. Specify optional message. 
		alias: bf
	backup restore HASH_ID 
		Restore the backup of hash id
		alias: br
	backup list [RECENT_MAX_ITEMS]
		List backups specifying number of recent items to list.
		alias: bl
	backup period INTERVAL_IN_SECONDS
		Set automatic backup perid. Set to 0 to disable.
		alias: bp
`)

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
		glog.Errorf("invalid command '%s'\n", parts[0])
		return nil
	}

	glog.Infof("Handler found, invoking")
	err = h.Handle(ctx, sm, parts)
	if err != nil {
		glog.Error(err)
	}
	return nil
}
