package svrmgr

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/golang/glog"
)

type LogLine struct {
	Line string
	Time time.Time
}

type Process struct {
	provider     Provider
	cmd          *exec.Cmd
	stdOut       io.ReadCloser
	stdErr       io.ReadCloser
	stdIn        io.WriteCloser
	stdoutLines  []LogLine
	outputReader chan string
}

func NewProcess(provider Provider, cmd *exec.Cmd) *Process {
	return &Process{
		provider: provider,
		cmd:      cmd,
	}
}

func (proc *Process) SendInput(line string) error {
	line += "\r\n"
	lineBytes := []byte(line)
	proc.provider.Log(fmt.Sprintf(">%s", line))
	n, err := proc.stdIn.Write(lineBytes)
	if n != len(lineBytes) {
		return fmt.Errorf("unable to write to bedrock server")
	}
	return err
}

func (proc *Process) StartReadOutput(c chan string) {
	proc.outputReader = c
}

func (proc *Process) EndReadOutput() {
	if proc.outputReader != nil {
		close(proc.outputReader)
		proc.outputReader = nil
	}
}

func (proc *Process) Start(ctx context.Context, provider Provider) error {
	var err error
	proc.stdOut, _ = proc.cmd.StdoutPipe()
	proc.stdErr, _ = proc.cmd.StderrPipe()
	proc.stdIn, _ = proc.cmd.StdinPipe()

	go func() {
		if err := proc.cmd.Start(); err != nil {
			provider.Log(fmt.Sprintf("unable to start bedrock server. %v", err))
		}
		go proc.handleStdOut(provider, proc.stdOut, true)
		go proc.handleStdOut(provider, proc.stdErr, false)
		err = proc.cmd.Wait()
		if err != nil {
			provider.Log(fmt.Sprintf("server exited with failure. %v", err))
		} else {
			provider.Log("server exited with success")
		}
		proc.EndReadOutput()
	}()
	return nil
}

func (proc *Process) IsRunning() bool {
	return proc.cmd != nil && proc.cmd.Process != nil && proc.cmd.ProcessState == nil
}

func (proc *Process) Kill() error {
	if proc.IsRunning() {
		glog.Infof("killing bedrock server")
		return proc.cmd.Process.Kill()
	}
	return nil
}

// Private functions
func (proc *Process) handleStdOut(provider Provider, stdOut io.ReadCloser, capture bool) {
	scanner := bufio.NewScanner(stdOut)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		proc.processOutputLine(provider, line)
		if capture {
			proc.stdoutLines = append(proc.stdoutLines, LogLine{Line: line, Time: time.Now()})
		}
		if proc.outputReader != nil {
			proc.outputReader <- line
		}
	}
	glog.Infof("scanner completed")
}

func (proc *Process) processOutputLine(provider Provider, line string) {
	provider.Log(line)
}
