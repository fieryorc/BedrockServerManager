package svrmgr

import (
	"context"
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

var gitExecutable = flag.String("git_exe", "git.exe", "path to the git executable (if git.exe is not in the PATH)")

// gitWrapper provides git functionality.
type gitWrapper struct {
	exe   string
	wsDir string
}

type GitReference struct {
	Ref  string
	Type string
}

type GitWrapper interface {
	RunCommand(ctx context.Context, args ...string) (string, error)
	IsDirClean(ctx context.Context) (bool, error)
	DeleteBranches(ctx context.Context, provider Provider, args []string) error
	GetCurrentHead(context.Context) (GitReference, error)
	Checkout(context.Context, GitReference) error
}

// newGitWrapper returns new instance of git wrapper.
func newGitWrapper(wsDir string) *gitWrapper {
	var err error
	exePath := *gitExecutable
	if !filepath.IsAbs(exePath) {
		exePath, err = exec.LookPath(*gitExecutable)
		if err != nil {
			panic("git executable not found")
		}
	}

	glog.Infof("git exe = %s, root = %s", exePath, wsDir)
	return &gitWrapper{
		exe:   exePath,
		wsDir: wsDir,
	}
}

// RunCommand runs git command and returs the results.
// Output is not printed to the console.
func (gw *gitWrapper) RunCommand(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, gw.exe, args...)
	cmd.Dir = gw.wsDir

	glog.Infof("running %s %s", cmd.Path, strings.Join(cmd.Args, " "))
	out, err := cmd.CombinedOutput()

	if err != nil {
		glog.Infof("command failed. %v", err)
		glog.Infof(string(out))
		return string(out), fmt.Errorf("failed to run %s %s. %v", gw.exe, strings.Join(args, " "), err)
	}

	if cmd.ProcessState.ExitCode() != 0 {
		glog.Infof("command failed with exit code %v", cmd.ProcessState.ExitCode())
		glog.Infof(string(out))
		return string(out), fmt.Errorf("git command failed with exit code %v", cmd.ProcessState.ExitCode())
	}

	return string(out), nil
}

// IsDirClean returns true if the git directory is clean.
func (gw *gitWrapper) IsDirClean(ctx context.Context) (bool, error) {
	ctxTimeout, _ := context.WithTimeout(ctx, *saveTimeout)
	out, err := gw.RunCommand(ctxTimeout, "status")
	glog.Info("git status:")
	glog.Infof(out)
	if err != nil {
		return false, err
	}
	return strings.Contains(out, "nothing to commit, working tree clean"), nil
}

func (gw *gitWrapper) DeleteBranches(ctx context.Context, provider Provider, args []string) error {
	var err error
	if len(args) == 0 {
		return fmt.Errorf("must specify at least one branch to delete")
	}

	ctxTimeout, _ := context.WithTimeout(ctx, *saveTimeout)
	cmdArgs := []string{
		"branch",
		"-av",
		"--format=%(refname:lstrip=2) %(contents:subject) %(if)%(HEAD)%(then)*%(end)",
		"--list",
	}
	cmdArgs = append(cmdArgs, args...)

	out, err := gw.RunCommand(ctxTimeout, cmdArgs...)
	if err != nil {
		return err
	}

	// Parse output to get the branch list
	var branches []string
	{
		lines := strings.Split(out, "\n")
		active := ""
		for _, l := range lines {
			if l == "" {
				continue
			}
			comps := strings.Split(l, " ")
			b := strings.Trim(comps[0], "\r\n\t ")
			if strings.Contains(l, "*") {
				active = b
			} else {
				branches = append(branches, b)
			}
		}

		if active != "" {
			if len(branches) == 0 {
				return fmt.Errorf("active backup %s cannot be deleted", active)
			} else {
				provider.Log(fmt.Sprintf("active branch '%s' cannot be deleted", active))
			}
		}
	}

	provider.Log(fmt.Sprintf("deleting the following backups:\r\n%s", strings.Join(branches, "\r\n")))
	ctxTimeout, _ = context.WithTimeout(ctx, *saveTimeout)
	cmdArgs = []string{
		"branch",
		"-D",
	}
	cmdArgs = append(cmdArgs, branches...)
	out, err = gw.RunCommand(ctxTimeout, cmdArgs...)
	if err != nil {
		provider.Log(fmt.Sprintf("git branch -D failed. %s", out))
		return err
	}

	return nil
}

func (gw *gitWrapper) GetCurrentHead(ctx context.Context) (GitReference, error) {
	ctxTimeout, _ := context.WithTimeout(ctx, *saveTimeout)
	cmdArgs := []string{
		"rev-parse",
		"HEAD",
	}

	out, err := gw.RunCommand(ctxTimeout, cmdArgs...)
	if err != nil {
		return GitReference{}, err
	}

	return GitReference{Ref: strings.Trim(out, "\r\n ")}, nil
}
func (gw *gitWrapper) Checkout(ctx context.Context, gr GitReference) error {
	ctxTimeout, _ := context.WithTimeout(ctx, *saveTimeout)
	cmdArgs := []string{
		"checkout",
		gr.Ref,
	}

	_, err := gw.RunCommand(ctxTimeout, cmdArgs...)
	if err != nil {
		return err
	}
	return nil
}
