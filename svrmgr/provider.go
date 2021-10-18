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

// Provider implements the common functionality required by
// plugins.
type Provider interface {
	Println(str string)
	Printf(format string, args ...interface{})
	Printfln(format string, args ...interface{})
	Log(line string)
	RunCommand(ctx context.Context, cmd string) error
	SetServerProcess(sp *exec.Cmd)
	GetServerProcess() *Process
	ResetServerProcess()
	GitWrapper() GitWrapper
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

func (sm *ServerManager) SetServerProcess(cmd *exec.Cmd) {
	sm.serverProcess.cmd = cmd
}

func (sm *ServerManager) GetServerProcess() *Process {
	return sm.serverProcess
}

func (sm *ServerManager) ResetServerProcess() {
	sm.serverProcess.cmd = nil
}

func (sm *ServerManager) GitWrapper() GitWrapper {
	return sm.gw
}
