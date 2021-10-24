package svrmgr

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fieryorc/BedrockServerManager/winutils"
	gomock "github.com/golang/mock/gomock"
)

func newBackupTest(t *testing.T) *svrmgrTest {
	return newSvrMgrTest(t)
}

func TestBackup_CleanServerNotRunning(t *testing.T) {
	st := newBackupTest(t)
	defer st.close(t)

	st.spMock.EXPECT().IsRunning().Return(false)
	st.spMock.EXPECT().Kill()
	st.gwMock.EXPECT().IsDirClean(gomock.Any()).Return(true, nil)

	st.PushCommandAsync("backup save test backup")
	st.PushCommandAsync("quit")

	err := st.sm.Process(context.Background(), []string{})
	if err != nil {
		t.Errorf("expecting nil, got %v", err)
	}
	exp := "skipping backup. no dirty files"
	if !strings.Contains(st.stdoutLog.String(), exp) {
		t.Errorf("expected: %s", exp)
	}
}

func TestBackup_CleanServerRunning(t *testing.T) {
	st := newBackupTest(t)
	defer st.close(t)
	var ch chan string

	st.spMock.EXPECT().IsRunning().Return(true)
	st.spMock.EXPECT().Kill()
	st.gwMock.EXPECT().IsDirClean(gomock.Any()).Return(true, nil)
	st.spMock.EXPECT().StartReadOutput(gomock.Any()).DoAndReturn(func(c chan string) {
		ch = c
	})
	st.spMock.EXPECT().EndReadOutput()
	st.spMock.EXPECT().SendInput(gomock.Any()).AnyTimes().DoAndReturn(func(inp string) error {
		if inp == "save hold" {
			ch <- winutils.AddNewLine(backupSaveCompletedMarker)
			ch <- winutils.AddNewLine("file list...")
		}
		return nil
	})

	st.PushCommandAsync("backup save test backup")
	st.PushCommandAsync("quit")

	err := st.sm.Process(context.Background(), []string{})
	if err != nil {
		t.Errorf("expecting nil, got %v", err)
	}
	exp := "skipping backup. no dirty files"
	if !strings.Contains(st.stdoutLog.String(), exp) {
		t.Errorf("expected: %s", exp)
	}
}

func TestBackup_BackupSimple(t *testing.T) {
	st := newBackupTest(t)
	defer st.close(t)
	var ch chan string

	st.spMock.EXPECT().IsRunning().Return(true)
	st.spMock.EXPECT().Kill()
	st.gwMock.EXPECT().IsDirClean(gomock.Any()).Return(false, nil)
	st.spMock.EXPECT().StartReadOutput(gomock.Any()).DoAndReturn(func(c chan string) {
		ch = c
	})
	st.spMock.EXPECT().EndReadOutput()
	st.spMock.EXPECT().SendInput(gomock.Any()).AnyTimes().DoAndReturn(func(inp string) error {
		if inp == "save hold" {
			ch <- winutils.AddNewLine(backupSaveCompletedMarker)
			ch <- winutils.AddNewLine("file list...")
		}
		return nil
	})
	st.gwMock.EXPECT().RunGitCommand(gomock.Any(), gomock.Any(), gomock.Any()).Return("complete", nil)
	st.gwMock.EXPECT().RunGitCommand(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("complete", nil)
	st.gwMock.EXPECT().RunGitCommand(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("complete", nil)

	st.PushCommandAsync("backup save test backup")
	st.PushCommandAsync("quit")

	err := st.sm.Process(context.Background(), []string{})
	if err != nil {
		t.Errorf("expecting nil, got %v", err)
	}
	exp := "backup success"
	if !strings.Contains(st.stdoutLog.String(), exp) {
		t.Errorf("expected: %s", exp)
	}
}

func TestClean_ServerRunning(t *testing.T) {
	st := newBackupTest(t)
	defer st.close(t)

	st.spMock.EXPECT().IsRunning().Return(true)
	st.spMock.EXPECT().Kill()

	st.PushCommandAsync("backup clean")
	st.PushCommandAsync("quit")

	err := st.sm.Process(context.Background(), []string{})
	if err != nil {
		t.Errorf("expecting nil, got %v", err)
	}
	exp := "cannot clean. server is running"
	if !strings.Contains(st.stdoutLog.String(), exp) {
		t.Errorf("expected: %s", exp)
	}
}

func TestClean_ServerNotRunning(t *testing.T) {
	st := newBackupTest(t)
	defer st.close(t)

	st.spMock.EXPECT().IsRunning().AnyTimes().Return(false)
	st.spMock.EXPECT().Kill()
	st.gwMock.EXPECT().IsDirClean(gomock.Any()).Return(false, nil)
	st.gwMock.EXPECT().GetCurrentHead(gomock.Any()).Return(GitReference{Ref: "refs/heads/foo"}, nil)
	st.gwMock.EXPECT().RunGitCommand(gomock.Any(), gomock.Any(), gomock.Any()).Return("complete", nil)
	st.gwMock.EXPECT().RunGitCommand(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("complete", nil)
	st.gwMock.EXPECT().RunGitCommand(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("complete", nil)
	st.gwMock.EXPECT().Checkout(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, ref GitReference) error {
		if ref.Ref != "refs/heads/foo" {
			t.Errorf("invalid ref receieved: %v", ref.Ref)
		}
		return nil
	})

	st.PushCommandAsync("backup clean")
	st.PushCommandAsync("quit")

	err := st.sm.Process(context.Background(), []string{})
	if err != nil {
		t.Errorf("expecting nil, got %v", err)
	}
	exp := "clean successful"
	if !strings.Contains(st.stdoutLog.String(), exp) {
		t.Errorf("expected: %s", exp)
	}
}

func TestList_Simple(t *testing.T) {
	st := newBackupTest(t)
	defer st.close(t)

	st.spMock.EXPECT().Kill()
	branchList := []GitReference{
		{
			Ref:  "refs/heads/1",
			Type: GitReferenceTypeBranch,
		},
		{
			Ref:    "refs/heads/2",
			Type:   GitReferenceTypeBranch,
			IsHead: true,
		},
	}
	st.gwMock.EXPECT().ListBranches(gomock.Any(), gomock.Any(), gomock.Any()).Return(branchList, nil)

	st.PushCommandAsync("backup list")
	st.PushCommandAsync("quit")

	err := st.sm.Process(context.Background(), []string{})
	if err != nil {
		t.Errorf("expecting nil, got %v", err)
	}
	exp := "refs/heads/1"
	if !strings.Contains(st.stdoutLog.String(), exp) {
		t.Errorf("expected: %s", exp)
	}
	exp = "* refs/heads/2"
	if !strings.Contains(st.stdoutLog.String(), exp) {
		t.Errorf("expected: %s", exp)
	}
}

func newTestBranch(prefix string, date time.Time) GitReference {
	return GitReference{
		Ref:                prefix + date.Local().Format(FormatBackupTimestamp),
		Type:               GitReferenceTypeBranch,
		CommitDate:         date,
		Hash:               "testhash",
		CommitDateRelative: "relative date",
	}
}
func TestPrune_Simple(t *testing.T) {
	st := newBackupTest(t)
	defer st.close(t)

	st.spMock.EXPECT().Kill()
	nowTime := time.Date(2021, 1, 2, 1, 0, 0, 0, time.UTC)
	st.nowFn = func() time.Time {
		return nowTime
	}
	branchList := []GitReference{
		// 1 day earlier
		newTestBranch("saves/periodic/", nowTime.Add(-time.Hour*24)),
		newTestBranch("saves/periodic/", nowTime.Add(-time.Hour*24).Add(-time.Hour)),

		// 2 day earlier
		newTestBranch("saves/periodic/", nowTime.Add(-time.Hour*48)),
	}
	for i := 1; i < 10; i++ {
		b := newTestBranch("saves/periodic/", nowTime.Add(-time.Hour*time.Duration(i)))
		b.IsHead = i == 5
		branchList = append(branchList, b)
	}

	st.gwMock.EXPECT().ListBranches(gomock.Any(), gomock.Any(), gomock.Any()).Return(branchList, nil)
	st.gwMock.EXPECT().DeleteBranches(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, prov Provider, refs []GitReference) error {
			for _, r := range refs {
				t.Logf("deleting branch %v", r)
			}
			if len(refs) != 2 {
				t.Errorf("incorrect number of branches to delete. Exp: 1, Got: %d", len(refs))
			}
			return nil
		})

	st.PushCommandAsync("backup prune 12h 12h")
	st.PushCommandAsync("quit")

	err := st.sm.Process(context.Background(), []string{})
	if err != nil {
		t.Errorf("expecting nil, got %v", err)
	}
}
