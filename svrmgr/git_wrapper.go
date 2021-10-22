package svrmgr

import (
	"context"
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
)

//go:generate mockgen -package svrmgr -source=git_wrapper.go -destination=git_wrapper_mocks_test.go

var gitExecutable = flag.String("git_exe", "git.exe", "path to the git executable (if git.exe is not in the PATH)")
var gitDryRun = flag.Bool("git_dry_run", false, "if specified, git update operations will not be performed")
var commandTimeout = flag.Duration("git_command_timeout", time.Second*30, "Time to wait for git command to complete")

// gitWrapper provides git functionality.
type gitWrapper struct {
	exe   string
	wsDir string
}

// GitReferenceType represents the type of the git reference.
type GitReferenceType string

const (
	GitReferenceTypeBranch GitReferenceType = "branch"
	GitReferenceTypeTag    GitReferenceType = "tag"
	GitReferenceTypeCommit GitReferenceType = "commit"
)

// GitReference type.
type GitReference struct {
	Ref                string
	Type               GitReferenceType
	IsHead             bool   // True if this is current head
	Hash               string // Commit hash
	Subject            string // Current commit subject line
	CommitDate         time.Time
	CommitDateRelative string
}

func (gr GitReference) String() string {
	active := " "
	if gr.IsHead {
		active = "*"
	}
	return fmt.Sprintf("%s %s %s %s (%s)", active, gr.Ref, gr.Hash, gr.Subject, gr.CommitDateRelative)
}

// GitWrapper provides wrapper for git.
type GitWrapper interface {
	RunGitCommand(ctx context.Context, args ...string) (string, error)
	IsDirClean(ctx context.Context) (bool, error)
	DeleteBranches(ctx context.Context, provider Provider, refs []GitReference) error
	GetCurrentHead(context.Context) (GitReference, error)
	Checkout(context.Context, GitReference) error
	ListBranches(ctx context.Context, provider Provider, filters []string) ([]GitReference, error)
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

// RunGitCommand runs git command and returs the results.
// Output is not printed to the console.
func (gw *gitWrapper) RunGitCommand(ctx context.Context, args ...string) (string, error) {
	ctxTimeout, _ := context.WithTimeout(ctx, *commandTimeout)
	cmd := exec.CommandContext(ctxTimeout, gw.exe, args...)
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
	out, err := gw.RunGitCommand(ctx, "status")
	glog.Info("git status:")
	glog.Infof(out)
	if err != nil {
		return false, err
	}

	return strings.Contains(out, "nothing to commit, working tree clean"), nil
}

func (gw *gitWrapper) GetCurrentHead(ctx context.Context) (GitReference, error) {
	cmdArgs := []string{
		"rev-parse",
		"HEAD",
	}

	out, err := gw.RunGitCommand(ctx, cmdArgs...)
	if err != nil {
		return GitReference{}, err
	}

	return GitReference{Ref: strings.Trim(out, "\r\n ")}, nil
}
func (gw *gitWrapper) Checkout(ctx context.Context, gr GitReference) error {
	cmdArgs := []string{
		"checkout",
		gr.Ref,
	}

	_, err := gw.RunGitCommand(ctx, cmdArgs...)
	if err != nil {
		return err
	}
	return nil
}

func (gw *gitWrapper) ListBranches(ctx context.Context, provider Provider, filters []string) ([]GitReference, error) {
	var result []GitReference
	var err error

	cmdArgs := []string{
		"branch",
		"-av",
		"--format=%(refname:lstrip=2)$XYX$%(objectname:short)$XYX$%(contents:subject)$XYX$%(committerdate)$XYX$%(committerdate:relative)$XYX$%(if)%(HEAD)%(then)*%(end)$XYX$",
		"--list",
	}
	if len(filters) > 0 {
		cmdArgs = append(cmdArgs, filters...)
	}

	out, err := gw.RunGitCommand(ctx, cmdArgs...)
	if err != nil {
		return nil, err
	}

	// Parse output to get the branch list

	lines := strings.Split(out, "\n")
	for _, l := range lines {
		if l == "" {
			continue
		}
		comps := strings.Split(l, "$XYX$")
		commitDate, err := time.Parse("Mon Jan 02 15:04:05 2006 -0700", comps[3])
		if err != nil {
			return nil, fmt.Errorf("invalid date from git.. internal error. %v", err)
		}
		result = append(result, GitReference{
			Ref:                strings.Trim(comps[0], "\r\n\t "),
			Hash:               strings.Trim(comps[1], "\r\n\t "),
			Type:               GitReferenceTypeBranch,
			Subject:            strings.Trim(comps[2], "\r\n\t "),
			CommitDateRelative: strings.Trim(comps[4], "\r\n\t "),
			IsHead:             comps[5] == "*",
			CommitDate:         commitDate,
		})
	}

	return result, nil
}

func (gw *gitWrapper) DeleteBranches(ctx context.Context, provider Provider, branches []GitReference) error {
	var err error
	if len(branches) == 0 {
		return fmt.Errorf("must specify at least one branch to delete")
	}

	// Print warning if deleting active branch.
	var logs []string
	var branchList []string
	for _, b := range branches {
		if b.IsHead {
			if len(branches) == 1 {
				return fmt.Errorf("active backup %s cannot be deleted", b.Ref)
			} else {
				provider.Log(fmt.Sprintf("active branch '%s' cannot be deleted", b.Ref))
			}
		} else {
			logs = append(logs, b.String())
			branchList = append(branchList, b.Ref)
		}
	}

	provider.Log(fmt.Sprintf("deleting the following backups:\r\n%s", strings.Join(logs, "\r\n")))
	cmdArgs := []string{
		"branch",
		"-D",
	}

	if *gitDryRun {
		provider.Log("*** dry run only. deletion not performed ****")
		return nil
	}

	cmdArgs = append(cmdArgs, branchList...)
	out, err := gw.RunGitCommand(ctx, cmdArgs...)
	if err != nil {
		provider.Log(fmt.Sprintf("git branch -D failed. %s", out))
		return err
	}

	return nil
}
