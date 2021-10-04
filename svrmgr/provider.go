package svrmgr

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

type Provider interface {
	Println(str string)
	Printf(format string, args ...interface{})
	Printfln(format string, args ...interface{})
	Log(line string)
	RunCommand(ctx context.Context, cmd string) error
	SetServerProcess(sp *exec.Cmd)
	GetServerProcess() *Process
	ResetServerProcess()
}

func (sm *ServerManager) Println(str string) {
	fmt.Println(str)
}

func (sm *ServerManager) Printf(format string, args ...interface{}) {
	fmt.Printf("%s", fmt.Sprintf(format, args...))
}

func (sm *ServerManager) Printfln(format string, args ...interface{}) {
	fmt.Printf("%s\r\n", fmt.Sprintf(format, args...))
}

// Log output to the console. Usually always visible, and includes timestamp
func (sm *ServerManager) Log(line string) {
	fmt.Printf("%s:%s\r\n", time.Now().Local().Format("20060102-15:04:05"), line)
}

func (sm *ServerManager) RunCommand(ctx context.Context, cmd string) error {
	return sm.handleCommand(ctx, cmd)
}

func (sm *ServerManager) SetServerProcess(sp *exec.Cmd) {
	sm.serverProcess = NewProcess(sm, sp)
}

func (sm *ServerManager) GetServerProcess() *Process {
	return sm.serverProcess
}

func (sm *ServerManager) ResetServerProcess() {
	sm.serverProcess = nil
}
