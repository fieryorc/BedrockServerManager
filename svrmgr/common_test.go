package svrmgr

import (
	"bytes"
	"flag"
	"io"
	"sync"
	"testing"
	"time"

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
	nowFn            func() time.Time
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
		sm:    newServerManagerForTests(),
		nowFn: time.Now,
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
	bh := st.sm.handlers["backup"].(*backupHandler)
	bh.nowFn = func() time.Time {
		return st.nowFn()
	}

	// Skip backup period output
	st.ReadOutputLine(t)

	return st
}
