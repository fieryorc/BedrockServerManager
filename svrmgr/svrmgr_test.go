package svrmgr

import (
	"bytes"
	"context"
	"flag"
	"io"
	"strings"
	"testing"

	"github.com/fieryorc/BedrockServerManager/winutils"
)

type svrmgrTest struct {
	sm           *ServerManager
	stdinReader  *io.PipeReader
	stdinWriter  *io.PipeWriter
	stdoutReader *io.PipeReader
	stdoutWriter *io.PipeWriter
	stdoutLog    bytes.Buffer
}

func (st *svrmgrTest) PushCommandAsync(str string) {
	go io.WriteString(st.stdinWriter, winutils.AddNewLine(str))
}

func (st *svrmgrTest) readStdout(t *testing.T) {
	buf := make([]byte, 1024)
	for {
		n, err := st.stdoutReader.Read(buf)
		if err != nil && err != io.EOF {
			t.Errorf("stdout read error. %v", err)
		}
		if n == 0 {
			break
		}

		if _, err := st.stdoutLog.Write(buf[:n]); err != nil {
			t.Errorf("unable to write to stdout log. %v", err)
		}
	}
	t.Log("stdout reader closed")
}

func (st *svrmgrTest) ReadOutputLine(t *testing.T) string {
	ln, err := st.stdoutLog.ReadString('\n')
	if err != nil {
		t.Fatalf("error reading from stdoutLog. %v", err)
	}
	return ln
}

func (st *svrmgrTest) close(t *testing.T) {
	st.stdinWriter.Close()
	st.stdinReader.Close()
	st.stdoutReader.Close()
	st.stdoutWriter.Close()
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
	err := st.sm.Process(context.Background(), []string{})
	if err != nil {
		t.Errorf("expecting nil, got %v", err)
	}
	ln := st.ReadOutputLine(t)
	if !strings.Contains(ln, "exiting the session") {
		t.Errorf("expected output not found. Found instead: %v", ln)
	}
}

func sTestProcess_Status(t *testing.T) {
	st := newSvrMgrTest(t)
	defer st.close(t)
	// TODO: need gitWrapper mock

	st.PushCommandAsync("status")
	st.PushCommandAsync("quit")
	err := st.sm.Process(context.Background(), []string{})
	if err != nil {
		t.Errorf("expecting nil, got %v", err)
	}
	ln := st.ReadOutputLine(t)
	if !strings.Contains(ln, "sdfsdf") {
		t.Errorf("expected output not found. Found instead: %v", ln)
	}
}
