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

// LogLine represents a single line of the log
type LogLine struct {
	Line string
	Time time.Time
}

// Process encapsulates the bedrock server running process.
type Process struct {
	provider     Provider
	cmd          *exec.Cmd
	stdOut       io.ReadCloser
	stdErr       io.ReadCloser
	stdIn        io.WriteCloser
	stdoutLines  []LogLine
	outputReader chan string // When set, the output is sent to this channel.
}

// NewProcess creates new process.
func NewProcess(provider Provider, cmd *exec.Cmd) *Process {
	return &Process{
		provider: provider,
		cmd:      cmd,
	}
}

// SendInput sends input to the running server.
func (proc *Process) SendInput(line string) error {
	if !proc.IsRunning() {
		return fmt.Errorf("server not running. cannot send input")
	}

	line += "\r\n"
	lineBytes := []byte(line)
	proc.provider.Log(fmt.Sprintf(">%s", line))
	n, err := proc.stdIn.Write(lineBytes)
	if n != len(lineBytes) {
		return fmt.Errorf("unable to write to bedrock server")
	}
	return err
}

// StartReadOutput sets the reader channel.
// All subsequent output from the server will be sent to this channel.
func (proc *Process) StartReadOutput(c chan string) {
	proc.outputReader = c
}

// EndReadOutput resets the output reader.
func (proc *Process) EndReadOutput() {
	if proc.outputReader != nil {
		close(proc.outputReader)
		proc.outputReader = nil
	}
}

// Start the server process.
func (proc *Process) Start(ctx context.Context, provider Provider) error {
	var err error

	if proc.IsRunning() {
		return fmt.Errorf("already running")
	}

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

// IsRunning returns true if the server is running.
func (proc *Process) IsRunning() bool {
	return proc.cmd != nil && proc.cmd.Process != nil && proc.cmd.ProcessState == nil
}

// Kill the running server process.
func (proc *Process) Kill() error {
	if proc.IsRunning() {
		glog.Infof("killing bedrock server")
		return proc.cmd.Process.Kill()
	}
	return nil
}

// handleStdOut should be run in its own go routine.
// Reads the server output and does the necessary processing.
// All server output is automatically printed to the console with timestamp.
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

// processOutputLine writes line to the console.
func (proc *Process) processOutputLine(provider Provider, line string) {
	provider.Log(line)
}
