package svrmgr

import (
	"bytes"
	"context"
	"flag"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/fieryorc/BedrockServerManager/winutils"
	gomock "github.com/golang/mock/gomock"
)

type svrmgrTest struct {
	ctrl             *gomock.Controller
	sm               *ServerManager
	stdinReader      *io.PipeReader
	stdinWriter      *io.PipeWriter
	stdoutReader     *io.PipeReader
	stdoutWriter     *io.PipeWriter
	stdoutLog        bytes.Buffer
	gwMock           *MockGitWrapper
	spMock           *MockServerProcess
	done             bool
	commandQueue     []string
	commandQueueLock sync.Mutex
}

func (st *svrmgrTest) processCommandQueue() {
	st.commandQueueLock.Lock()
	defer st.commandQueueLock.Unlock()
	for _, cmd := range st.commandQueue {
		io.WriteString(st.stdinWriter, winutils.AddNewLine(cmd))
	}
	st.commandQueue = nil
}

func (st *svrmgrTest) PushCommandAsync(str string) {
	st.commandQueueLock.Lock()
	defer st.commandQueueLock.Unlock()

	st.commandQueue = append(st.commandQueue, str)
	if len(st.commandQueue) == 1 {
		go st.processCommandQueue()
	}
}

func (st *svrmgrTest) readStdout(t *testing.T) {
	buf := make([]byte, 1024)
	for {
		n, err := st.stdoutReader.Read(buf)
		if err != nil && err != io.EOF && !st.done {
			t.Logf("stdout read error. %v", err)
		}
		if n == 0 {
			break
		}

		if _, err := st.stdoutLog.Write(buf[:n]); err != nil {
			t.Logf("unable to write to stdout log. %v", err)
		}
	}
}

func (st *svrmgrTest) ReadOutputLine(t *testing.T) string {
	ln, err := st.stdoutLog.ReadString('\n')
	if err != nil && !st.done {
		t.Logf("error reading from stdoutLog. %v", err)
	}
	return ln
}

func (st *svrmgrTest) close(t *testing.T) {
	st.done = true
	st.stdinWriter.Close()
	st.stdinReader.Close()
	st.stdoutReader.Close()
	st.stdoutWriter.Close()
	st.ctrl.Finish()
}

func newSvrMgrTest(t *testing.T) *svrmgrTest {
	flag.Set("bedrock_exe", `c:\foo.exe`)
	st := &svrmgrTest{
		sm: newServerManagerForTests(),
	}
	st.stdinReader, st.stdinWriter = io.Pipe()
	st.sm.stdin = st.stdinReader
	st.stdoutReader, st.stdoutWriter = io.Pipe()
	st.sm.stdout = st.stdoutWriter
	st.ctrl = gomock.NewController(t)
	st.gwMock = NewMockGitWrapper(st.ctrl)
	st.sm.gw = st.gwMock
	st.spMock = NewMockServerProcess(st.ctrl)
	st.sm.serverProcess = st.spMock
	go st.readStdout(t)
	st.sm.loadPlugings()
	// Skip backup period output
	st.ReadOutputLine(t)
	return st
}

func TestProcess_Startup(t *testing.T) {
	sm := newServerManagerForTests()
	if sm == nil {
		t.Errorf("failed to create server manager")
	}
}

func TestProcess_Basic(t *testing.T) {
	st := newSvrMgrTest(t)
	defer st.close(t)
	t.Logf("starting process")
	st.PushCommandAsync("quit")
	t.Logf("starting process")
	st.spMock.EXPECT().Kill()

	err := st.sm.Process(context.Background(), []string{})
	if err != nil {
		t.Errorf("expecting nil, got %v", err)
	}
	exp := "exiting the session"
	if !strings.Contains(st.stdoutLog.String(), exp) {
		t.Errorf("expected: %v", exp)
	}
}

func TestProcess_Status(t *testing.T) {
	st := newSvrMgrTest(t)
	defer st.close(t)

	st.gwMock.EXPECT().IsDirClean(gomock.Any()).Return(true, nil)
	st.spMock.EXPECT().IsRunning().Return(false)
	st.spMock.EXPECT().Kill()

	st.PushCommandAsync("status")
	st.PushCommandAsync("quit")
	err := st.sm.Process(context.Background(), []string{})
	if err != nil {
		t.Errorf("expecting nil, got %v", err)
	}
	exp := "server is not running, workspace is clean, automatic backup interval: 30m0s"
	if !strings.Contains(st.stdoutLog.String(), exp) {
		t.Errorf("expected: %s, Got: %v", exp, st.stdoutLog.String())
	}
}

func TestProcess_StartServerAlreadyRunning(t *testing.T) {
	st := newSvrMgrTest(t)
	defer st.close(t)

	st.spMock.EXPECT().IsRunning().Return(true)
	st.spMock.EXPECT().Kill()

	st.PushCommandAsync("start")
	st.PushCommandAsync("quit")
	err := st.sm.Process(context.Background(), []string{})
	if err != nil {
		t.Errorf("expecting nil, got %v", err)
	}
	exp := "server already running"
	if !strings.Contains(st.stdoutLog.String(), exp) {
		t.Errorf("expected: %s", exp)
	}
}

func TestProcess_StartServer(t *testing.T) {
	st := newSvrMgrTest(t)
	defer st.close(t)

	var ch chan string
	st.spMock.EXPECT().IsRunning().Return(false)
	st.spMock.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil)
	st.spMock.EXPECT().SetCmd(gomock.Any())
	st.spMock.EXPECT().StartReadOutput(gomock.Any()).DoAndReturn(func(c chan string) {
		ch = c
		go func() {
			t.Logf("writing server output marker")
			ch <- winutils.AddNewLine(serverOutputMarker)
			ch <- winutils.AddNewLine(serverOutputMarker)
			t.Logf("finished writing server output marker")
		}()
	})
	st.spMock.EXPECT().EndReadOutput()
	st.spMock.EXPECT().Kill()

	st.PushCommandAsync("start")
	st.PushCommandAsync("quit")

	err := st.sm.Process(context.Background(), []string{})
	if err != nil {
		t.Errorf("expecting nil, got %v", err)
	}
	exp := "exiting the session"
	if !strings.Contains(st.stdoutLog.String(), exp) {
		t.Errorf("expected: %s", exp)
	}
}
