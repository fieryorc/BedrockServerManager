package svrmgr

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/fieryorc/BedrockServerManager/winutils"
	"github.com/golang/glog"
)

//go:generate mockgen -package svrmgr -source=provider.go -destination=provider_mocks_test.go

// Provider implements the common functionality required by
// plugins.
type Provider interface {
	// Register plugin handler
	Register(cmd string, handler Handler)
	// Println prints bare output to consle.
	Println(str string)
	// Printf prints bare output to consle.
	Printf(format string, args ...interface{})
	// Printfln prints bare output to consle.
	Printfln(format string, args ...interface{})
	// Log output to the console. Prints timestamp along with it.
	Log(line string)
	// RunCommand runs the command.
	RunCommand(ctx context.Context, cmd string) error
	// InitServer initializes the bedrock server wrapper.
	InitServer(ctx context.Context, path, cwd string, args []string) ServerProcess
	// GetServerProcess returns the server process wrapper.
	GetServerProcess() ServerProcess
	// GitWrapper returns the wrapper for git.
	GitWrapper() GitWrapper
	// GetHandler returns the handler for the command.
	GetHandler(cmd string) (Handler, error)
}

// Register a handler for given command.
func (sm *ServerManager) Register(cmd string, handler Handler) {
	glog.Infof("Registering handler for %s", cmd)
	sm.handlers[cmd] = handler
}

func (sm *ServerManager) Println(str string) {
	glog.Infof("OUT: %s", str)
	io.WriteString(sm.stdout, fmt.Sprintln(str))
}

func (sm *ServerManager) Printf(format string, args ...interface{}) {
	glog.Infof("OUT: %s", fmt.Sprintf(format, args...))
	io.WriteString(sm.stdout, fmt.Sprintf(format, args...))
}

func (sm *ServerManager) Printfln(format string, args ...interface{}) {
	glog.Infof("OUT: %s\r\n", fmt.Sprintf(format, args...))
	io.WriteString(sm.stdout, winutils.AddNewLine(fmt.Sprintf(format, args...)))
}

// Log output to the console. Usually always visible, and includes timestamp
func (sm *ServerManager) Log(line string) {
	glog.Infof("OUT: [%s] %s\r\n", time.Now().Local().Format("20060102-15:04:05"), line)
	io.WriteString(sm.stdout, winutils.AddNewLine(fmt.Sprintf("[%s] %s", time.Now().Local().Format("20060102-15:04:05"), line)))
}

func (sm *ServerManager) RunCommand(ctx context.Context, cmd string) error {
	return sm.handleCommand(ctx, cmd)
}

func (sm *ServerManager) InitServer(ctx context.Context, path, dir string, args []string) ServerProcess {
	cmd := exec.CommandContext(ctx, path)
	cmd.Dir = dir
	sm.serverProcess.SetCmd(cmd)
	return sm.serverProcess
}

func (sm *ServerManager) GetServerProcess() ServerProcess {
	return sm.serverProcess
}

func (sm *ServerManager) GitWrapper() GitWrapper {
	return sm.gw
}

func (sm *ServerManager) GetHandler(name string) (Handler, error) {
	h, ok := sm.handlers[name]
	if !ok {
		return nil, fmt.Errorf("handler %v not found", name)
	}
	return h, nil
}
