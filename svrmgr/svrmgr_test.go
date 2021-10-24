package svrmgr

import (
	"context"
	"strings"
	"testing"

	"github.com/fieryorc/BedrockServerManager/winutils"
	gomock "github.com/golang/mock/gomock"
)

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
