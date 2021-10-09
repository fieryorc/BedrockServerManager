package svrmgr

import (
	"context"
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
)

var gitExecutable = flag.String("git_exe", "git.exe", "path to the git executable (if git.exe is not in the PATH)")
var saveTimeout = flag.Duration("save_timeout", time.Second*30, "Time to wait for save command to complete")
var gitRoot = flag.String("git_root", "", "git root directory for the world. If not specified, uses bedrock server directory")
var autoBackupInterval = flag.Duration("backup_interval", time.Minute*30, "automatic backup interval.")

// backupHandler handles the backup logic.
// Supports multiple sub commands.
type backupHandler struct {
	gitPath string
	gitRoot string
	lock    sync.Mutex  // All operations are atomic.
	timer   *time.Timer // Periodic backup timer
}

// initBackupHandler initializes the backup plugin and starts the
// periodic backup.
func initBackupHandler(prov Provider) {
	var err error
	exePath := *gitExecutable
	if !filepath.IsAbs(exePath) {
		exePath, err = exec.LookPath(*gitExecutable)
		if err != nil {
			panic("git executable not found")
		}
	}

	bedrockPath := getBedrockServerPath()
	glog.Infof("bedrock server path = %s, dir = %s", bedrockPath, filepath.Dir(bedrockPath))
	bh := &backupHandler{
		gitPath: exePath,
		gitRoot: filepath.Dir(bedrockPath),
		timer:   time.NewTimer(*autoBackupInterval),
	}
	if *gitRoot != "" {
		bh.gitRoot = *gitRoot
	}

	glog.Infof("Registering backup handler: %+v", *bh)
	Register("backup", bh)
	go bh.runBackupLoop(context.Background(), prov)
}

// Handle handles the main logic.
func (h *backupHandler) Handle(ctx context.Context, provider Provider, cmd []string) error {
	if len(cmd) < 2 {
		return fmt.Errorf("invalid command. try help")
	}
	switch cmd[1] {
	case "save":
		msg := strings.Join(cmd[2:], " ")
		if msg == "" {
			msg = "Manual save"
		}
		return h.Save(ctx, provider, msg)
	case "list":
		return h.List(ctx, provider, cmd[2:])
	case "period":
		return h.SetPeriod(ctx, provider, cmd[2:])
	case "restore":
		return h.Restore(ctx, provider, cmd[2:])
	case "clean":
		return h.Clean(ctx, provider, cmd[2:])
	default:
		return fmt.Errorf("unknown command. try help")
	}
}

// Save the backup using git.
// If the server is running, then issue `save hold` and then run backup.
// Once the backup is complete, issue `save resume`.
func (h *backupHandler) Save(ctx context.Context, provider Provider, msg string) error {
	var err error
	ch := make(chan string, 10)

	h.lock.Lock()
	defer h.lock.Unlock()

	if !provider.GetServerProcess().IsRunning() {
		return h.backupWithGit(ctx, provider, "", msg)
	}

	provider.GetServerProcess().StartReadOutput(ch)
	defer provider.GetServerProcess().EndReadOutput()

	if err = provider.GetServerProcess().SendInput("save hold"); err != nil {
		return fmt.Errorf("unable to communicate with bedrock server. %v", err)
	}
	defer provider.GetServerProcess().SendInput("save resume")
	time.Sleep(time.Millisecond * 250)

	// Wait till server is ready for copy.
	timeout, _ := context.WithTimeout(ctx, *saveTimeout)
	for {
		select {
		// Read the data from channel until we get ready message.
		case l := <-ch:
			glog.Infof("got from channel: %v", l)
			if strings.Contains(l, "Data saved. Files are now ready to be copied") {
				// Read the next line. This is the list of files
				l = <-ch
				if err = h.backupWithGit(ctx, provider, l, msg); err != nil {
					return err
				}
				return nil
			}
		case <-timeout.Done():
			return fmt.Errorf("timed out waiting for server. bailing out")
		default:
			glog.Infof("waiting for save to be ready")
			if err = provider.GetServerProcess().SendInput("save query"); err != nil {
				return fmt.Errorf("unable to communicate with bedrock server. %v", err)
			}
			time.Sleep(time.Millisecond * 500)
		}
	}
}

// Restore from backup.
// To restore, working directory must be clean and server must NOT be running.
func (h *backupHandler) Restore(ctx context.Context, provider Provider, args []string) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	if len(args) != 1 {
		return fmt.Errorf("invalid args. Must specify HASH to restore. try help for syntax")
	}
	gitHash := args[0]

	if provider.GetServerProcess().IsRunning() {
		return fmt.Errorf("stop the server before restoring the backup")
	}

	isClean, err := h.isDirClean(ctx)
	if err != nil {
		return err
	}
	if !isClean {
		return fmt.Errorf("there are dirty files in the directory. run 'backup save' or 'backup clean' to clean up")
	}

	ctxTimeout, _ := context.WithTimeout(ctx, *saveTimeout)
	_, err = h.runCommand(ctxTimeout, h.gitPath, "reset", "--hard", gitHash)
	if err != nil {
		return err
	}

	provider.Log(fmt.Sprintf("successfully restored to %s", gitHash))
	return nil
}

// List recent backups.
// Optionally accepts the max item count.
func (h *backupHandler) List(ctx context.Context, provider Provider, args []string) error {
	h.lock.Lock()
	defer h.lock.Unlock()
	var err error
	maxItems := 15

	if len(args) > 0 {
		maxItems, err = strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid max items. %v. try 'help' for usage", maxItems)
		}
	}

	ctxTimeout, _ := context.WithTimeout(ctx, *saveTimeout)
	cmdArgs := []string{
		"log",
		`--format=%h %s (%ad) %d`,
		"--decorate",
		"--date=relative",
		fmt.Sprintf("-%d", maxItems),
	}
	out, err := h.runCommand(ctxTimeout, h.gitPath, cmdArgs...)
	provider.Printfln("%s", out)
	if err != nil {
		return err
	}

	return nil
}

// SetPeriod sets backup interval for periodic backup.
func (h *backupHandler) SetPeriod(ctx context.Context, provider Provider, args []string) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	if len(args) != 1 {
		return fmt.Errorf("invalid args. must specify INTERVAL. try 'help' for usage")
	}

	duration := args[0]
	interval, err := time.ParseDuration(duration)
	if err != nil {
		return fmt.Errorf("failed to set backup interval. %v", err)
	}
	if interval > 0 && interval < time.Second {
		return fmt.Errorf("backup period cannot be shorter than a second")
	}
	provider.Log(fmt.Sprintf("backup interval set to %v", interval))
	*autoBackupInterval = interval
	if !h.timer.Stop() {
		<-h.timer.C
	}
	h.timer.Reset(interval)
	return nil
}

// Clean the working directory by discarding the files.
func (h *backupHandler) Clean(ctx context.Context, provider Provider, args []string) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	if provider.GetServerProcess().IsRunning() {
		return fmt.Errorf("cannot clean. server is running")
	}

	ctxTimeout, _ := context.WithTimeout(ctx, *saveTimeout)
	_, err := h.runCommand(ctxTimeout, h.gitPath, "clean", "-df", "worlds")
	if err != nil {
		return fmt.Errorf("failed to clean. %v", err)
	}
	_, err = h.runCommand(ctxTimeout, h.gitPath, "reset", "--hard")
	if err != nil {
		return fmt.Errorf("failed to reset. %v", err)
	}
	provider.Log("clean successful")
	return nil
}

// backupWithGit implements the backup logic.
func (h *backupHandler) backupWithGit(ctx context.Context, provider Provider, fileList string, description string) error {
	var err error

	isClean, err := h.isDirClean(ctx)
	if err != nil {
		return err
	}
	if isClean {
		provider.Log("skipping backup. no dirty files")
		return nil
	}

	ctxTimeout, _ := context.WithTimeout(ctx, *saveTimeout)
	out, err := h.runCommand(ctxTimeout, h.gitPath, "add", "worlds")
	if err != nil {
		provider.Log(out)
		return err
	}

	ctxTimeout, _ = context.WithTimeout(ctx, *saveTimeout)

	if description == "" {
		panic("backup description not set")
	}

	out, err = h.runCommand(ctxTimeout, h.gitPath, "commit", "--allow-empty", "-m", description)
	if err != nil {
		provider.Log(out)
		provider.Log(fmt.Sprintf("backup failed. %v", err))
		return err
	}
	provider.Log("backup success")
	return nil

}

// runCommand runs git command and returs the results.
// Output is not printed to the console.
func (h *backupHandler) runCommand(ctx context.Context, path string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = h.gitRoot

	glog.Infof("running %s %s", cmd.Path, strings.Join(cmd.Args, " "))
	out, err := cmd.CombinedOutput()

	if err != nil {
		glog.Infof("command failed. %v", err)
		glog.Infof(string(out))
		return string(out), fmt.Errorf("failed to run %s %s. %v", path, strings.Join(args, " "), err)
	}

	if cmd.ProcessState.ExitCode() != 0 {
		glog.Infof("command failed with exit code %v", cmd.ProcessState.ExitCode())
		glog.Infof(string(out))
		return string(out), fmt.Errorf("git command failed with exit code %v", cmd.ProcessState.ExitCode())
	}

	return string(out), nil
}

// runBackupLoop runs the main backup loop.
func (h *backupHandler) runBackupLoop(ctx context.Context, prov Provider) {
	for {
		_, more := <-h.timer.C
		h.Save(context.Background(), prov, "Automatic periodic backup")
		if !more {
			// Channel closed.
			glog.Infof("periodic backup ending")
			return
		}
		// Restart timer
		h.timer.Reset(*autoBackupInterval)
	}
}

// isDirClean returns true if the git directory is clean.
func (h *backupHandler) isDirClean(ctx context.Context) (bool, error) {
	ctxTimeout, _ := context.WithTimeout(ctx, *saveTimeout)
	out, err := h.runCommand(ctxTimeout, h.gitPath, "status")
	glog.Info("git status:")
	glog.Infof(out)
	if err != nil {
		return false, err
	}
	return strings.Contains(out, "nothing to commit, working tree clean"), nil
}
