package svrmgr

import (
	"context"
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
)

var gitExecutable = flag.String("git_exe", "git.exe", "path to the git executable (if git.exe is not in the PATH)")
var saveTimeout = flag.Duration("save_timeout", time.Second*30, "Time to wait for save command to complete")
var gitRoot = flag.String("git_root", "", "git root directory for the world. If not specified, uses bedrock server directory")

type backupHandler struct {
	gitPath string
	gitRoot string
	lock    sync.Mutex
}

func init() {
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
	}
	if *gitRoot != "" {
		bh.gitRoot = *gitRoot
	}

	glog.Infof("Registering backup handler: %+v", *bh)
	Register("backup", bh)
}

func (h *backupHandler) Handle(ctx context.Context, provider Provider, cmd []string) error {
	if len(cmd) < 2 {
		return fmt.Errorf("invalid command. try help")
	}
	switch cmd[1] {
	case "save":
		return h.save(ctx, provider, strings.Join(cmd[2:], " "))
	default:
		return fmt.Errorf("unknown command. try help")
	}
}

func (h *backupHandler) save(ctx context.Context, provider Provider, msg string) error {
	var err error
	ch := make(chan string, 10)

	h.lock.Lock()
	defer h.lock.Unlock()

	provider.GetServerProcess().StartReadOutput(ch)
	defer provider.GetServerProcess().EndReadOutput()

	if err = provider.GetServerProcess().SendInput("save hold"); err != nil {
		return fmt.Errorf("unable to communicate with bedrock server. %v", err)
	}
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
				if err = provider.GetServerProcess().SendInput("save resume"); err != nil {
					return err
				}
				return nil
			}
		case <-timeout.Done():
			glog.Errorf("timed out. bailing out")
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

func (h *backupHandler) restore(hash string) error {
	return nil
}

func (h *backupHandler) list() error {
	return nil
}

func (h *backupHandler) setPeriod(period string) error {
	return nil
}

func (h *backupHandler) backupWithGit(ctx context.Context, provider Provider, fileList string, description string) error {
	var err error
	provider.Log(fmt.Sprintf("running git to backup. %s", fileList))

	ctxTimeout, _ := context.WithTimeout(ctx, *saveTimeout)
	if err = h.runCommand(ctxTimeout, h.gitPath, "add", "worlds"); err != nil {
		return err
	}

	ctxTimeout, _ = context.WithTimeout(ctx, *saveTimeout)

	if description == "" {
		description = fmt.Sprintf("%s Bedrock Auto Save", time.Now().Local().Format("20060102-15:04:05"))
	}

	if err = h.runCommand(ctxTimeout, h.gitPath, "commit", "-m", description); err != nil {
		provider.Log(fmt.Sprintf("backup failed. %v", err))
		return err
	}
	provider.Log("backup success")
	return nil

}

func (h *backupHandler) runCommand(ctx context.Context, path string, args ...string) error {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = h.gitRoot

	out, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("git add failed. %v", err)
		glog.Error(string(out))
		return fmt.Errorf("failed to run %s %s. %v", path, strings.Join(args, " "), err)
	}
	return nil
}
