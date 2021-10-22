package svrmgr

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
)

var autoBackupInterval = flag.Duration("backup_interval", time.Minute*30, "automatic backup interval.")

type backupType string

const (
	// backupTypeManual - manual backups
	backupTypeManual backupType = "manual"
	// backupTypePeriodic - Periodic backups
	backupTypePeriodic backupType = "periodic"
	// backupTypeTemp - temporary saves as a result of running clean.
	backupTypeTemp backupType = "temp"
)

// backupHandler handles the backup logic.
// Supports multiple sub commands.
type backupHandler struct {
	lock           sync.Mutex    // All operations are atomic.
	timer          *time.Timer   // Periodic backup timer
	backupInterval time.Duration // Automatic backup interval.
}

// initBackupHandler initializes the backup plugin and starts the
// periodic backup.
func initBackupHandler(prov Provider) {
	bh := &backupHandler{
		timer: time.NewTimer(time.Hour), // Will be reset immediately.
	}
	bh.setPeriod(context.Background(), prov, *autoBackupInterval)

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
			return fmt.Errorf("backup description must be specified")
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
	case "delete":
		return h.Delete(ctx, provider, cmd[2:])
	case "prune":
		return h.Prune(ctx, provider, cmd[2:])
	default:
		return fmt.Errorf("unknown command. try help")
	}
}

// Save the backup using git.
// If the server is running, then issue `save hold` and then run backup.
// Once the backup is complete, issue `save resume`.
func (h *backupHandler) Save(ctx context.Context, provider Provider, msg string) error {

	h.lock.Lock()
	defer h.lock.Unlock()
	return h.save(ctx, provider, backupTypeManual, msg)
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

	isClean, err := provider.GitWrapper().IsDirClean(ctx)
	if err != nil {
		return err
	}
	if !isClean {
		return fmt.Errorf("there are changes since last backup. run 'backup save' or 'backup clean' to clean up")
	}

	ctxTimeout, _ := context.WithTimeout(ctx, *commandTimeout)
	_, err = provider.GitWrapper().RunGitCommand(ctxTimeout, "checkout", gitHash)
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

	branches, err := provider.GitWrapper().ListBranches(ctx, provider, args)
	if err != nil {
		return err
	}

	var branchList []string
	for _, b := range branches {
		branchList = append(branchList, b.String())
	}
	provider.Printfln("%s", strings.Join(branchList, "\r\n"))
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

	interval, err := parseDuration(args[0])
	if err != nil {
		return fmt.Errorf("failed to set backup interval. %v", err)
	}

	return h.setPeriod(ctx, provider, interval)
}

// Clean the working directory by discarding the files.
func (h *backupHandler) Clean(ctx context.Context, provider Provider, args []string) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	if provider.GetServerProcess().IsRunning() {
		return fmt.Errorf("cannot clean. server is running")
	}

	curCommit, err := provider.GitWrapper().GetCurrentHead(ctx)
	if err != nil {
		return fmt.Errorf("internal failure. %v", err)
	}

	if err = h.save(ctx, provider, backupTypeTemp, "Saving for cleaning"); err != nil {
		return fmt.Errorf("failed to backup current contents. %v", err)
	}

	if err = provider.GitWrapper().Checkout(ctx, curCommit); err != nil {
		return fmt.Errorf("failed to restore old contents. %v", err)
	}

	provider.Log("clean successful")
	return nil
}

// Delete the specified backup
// Wildcards are not supported
func (h *backupHandler) Delete(ctx context.Context, provider Provider, args []string) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	var err error
	if len(args) == 0 {
		return fmt.Errorf("must specify at least one branch to delete")
	}

	branches, err := provider.GitWrapper().ListBranches(ctx, provider, args)
	if err != nil {
		return err
	}

	return provider.GitWrapper().DeleteBranches(ctx, provider, branches)
}

// Prune the backups
func (h *backupHandler) Prune(ctx context.Context, provider Provider, args []string) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	if len(args) != 2 {
		return fmt.Errorf("invalid arguments. try 'help'")
	}
	startTime, err := parseDuration(args[0])
	if err != nil {
		return fmt.Errorf("invalid start time. %v", err)
	}

	pruneInterval, err := parseDuration(args[1])
	if err != nil {
		return fmt.Errorf("invalid interval. %v", err)
	}

	branches, err := provider.GitWrapper().ListBranches(ctx, provider, []string{"saves/periodic/*"})
	if err != nil {
		return err
	}
	// Sort by ascending order
	sort.Slice(branches, func(i, j int) bool {
		return branches[i].CommitDate.Before(branches[j].CommitDate)
	})
	prunedBranches, err := h.getDeletionCanditates(ctx, provider, branches, startTime, pruneInterval)
	if err != nil {
		return err
	}
	return provider.GitWrapper().DeleteBranches(ctx, provider, prunedBranches)
}

// Status returns the current backup status.
// Called by other modules.
func (h *backupHandler) Status(ctx context.Context, provider Provider) string {
	output := ""
	if h.backupInterval == 0 {
		output += "automatic backup disabled"
	} else {
		output += fmt.Sprintf("automatic backup interval: %v", h.backupInterval)
	}

	return output
}

func (h *backupHandler) save(ctx context.Context, provider Provider, bt backupType, msg string) error {
	var err error
	ch := make(chan string, 10)

	if !provider.GetServerProcess().IsRunning() {
		return h.backupWithGit(ctx, provider, bt, msg)
	}

	provider.GetServerProcess().StartReadOutput(ch)
	defer provider.GetServerProcess().EndReadOutput()

	if err = provider.GetServerProcess().SendInput("save hold"); err != nil {
		return fmt.Errorf("unable to communicate with bedrock server. %v", err)
	}
	defer provider.GetServerProcess().SendInput("save resume")
	time.Sleep(time.Millisecond * 250)

	// Wait till server is ready for copy.
	timeout, _ := context.WithTimeout(ctx, *commandTimeout)
	for {
		select {
		// Read the data from channel until we get ready message.
		case l := <-ch:
			glog.Infof("got from channel: %v", l)
			if strings.Contains(l, "Data saved. Files are now ready to be copied") {
				// Read the next line. This is the list of files
				<-ch
				if err = h.backupWithGit(ctx, provider, bt, msg); err != nil {
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

// backupWithGit implements the backup logic.
func (h *backupHandler) backupWithGit(ctx context.Context, provider Provider, bt backupType, description string) error {
	var err error

	isClean, err := provider.GitWrapper().IsDirClean(ctx)
	if err != nil {
		return err
	}
	if isClean {
		provider.Log("skipping backup. no dirty files")
		return nil
	}

	ctxTimeout, _ := context.WithTimeout(ctx, *commandTimeout)
	out, err := provider.GitWrapper().RunGitCommand(ctxTimeout, "add", ".")
	if err != nil {
		provider.Log(out)
		return err
	}

	ctxTimeout, _ = context.WithTimeout(ctx, *commandTimeout)
	if description == "" {
		panic("backup description not set")
	}

	branch := fmt.Sprintf("saves/%s/%s", bt, time.Now().Local().Format("20060102-150405"))
	out, err = provider.GitWrapper().RunGitCommand(ctxTimeout, "checkout", "--orphan", branch)
	if err != nil {
		provider.Log(out)
		provider.Log(fmt.Sprintf("backup failed. %v", err))
		return err
	}

	out, err = provider.GitWrapper().RunGitCommand(ctxTimeout, "commit", "--allow-empty", "-m", description)
	if err != nil {
		provider.Log(out)
		provider.Log(fmt.Sprintf("backup failed. %v", err))
		return err
	}
	provider.Log("backup success")
	return nil

}

// SetPeriod sets backup interval for periodic backup.
func (h *backupHandler) setPeriod(ctx context.Context, provider Provider, interval time.Duration) error {
	if interval < 0 {
		return fmt.Errorf("backup period cannot be negative")
	}

	if interval > 0 && interval < time.Second {
		return fmt.Errorf("backup period cannot be shorter than a second")
	}

	if !h.timer.Stop() {
		glog.Infof("waiting for timer to drain")
		<-h.timer.C
	}
	h.backupInterval = interval
	if interval > 0 {
		provider.Log(fmt.Sprintf("backup interval set to %v", interval))
		h.timer.Reset(interval)
	} else {
		provider.Log("periodic backup suspended")
	}
	return nil
}

func (h *backupHandler) periodicBackup(ctx context.Context, provider Provider) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	return h.save(ctx, provider, backupTypePeriodic, "Automatic periodic backup")
}

// runBackupLoop runs the main backup loop.
func (h *backupHandler) runBackupLoop(ctx context.Context, prov Provider) {
	for {
		_, more := <-h.timer.C
		h.periodicBackup(context.Background(), prov)
		if !more {
			// Channel closed.
			glog.Infof("periodic backup ending")
			return
		}
		// Restart timer
		h.timer.Reset(h.backupInterval)
	}
}

func (h *backupHandler) getDeletionCanditates(
	ctx context.Context,
	provider Provider,
	branches []GitReference,
	cutoffTime time.Duration,
	pruneInterval time.Duration) ([]GitReference, error) {

	if len(branches) == 0 {
		return branches, nil
	}

	var result []GitReference
	cutOffDate := time.Now().Add(-cutoffTime)
	lastSkippedBranch := branches[0]
	for _, gr := range branches {
		glog.Infof("processing: %v", gr)

		// First skip all refs that are newer than startTime.
		if gr.CommitDate.After(cutOffDate) {
			lastSkippedBranch = gr
			continue
		}

		if gr.CommitDate.Sub(lastSkippedBranch.CommitDate) > pruneInterval {
			lastSkippedBranch = gr
			continue
		}

		result = append(result, gr)
	}

	return result, nil
}

func parseDuration(str string) (time.Duration, error) {
	if strings.HasSuffix(str, "d") {
		v, err := strconv.Atoi(str[:len(str)-1])
		if err != nil {
			return 0, err
		}
		return time.Duration(v) * time.Hour * 24, nil
	}

	return time.ParseDuration(str)
}
