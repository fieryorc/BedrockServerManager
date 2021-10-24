package svrmgr

import (
	"context"
	"fmt"
)

// helpHandler implements help command.
type helpHandler struct{}

func initHelpHandler(provider Provider) {
	provider.Register("help", &helpHandler{})
}

func (h *helpHandler) Handle(ctx context.Context, provider Provider, cmd []string) error {
	fmt.Printf(`Welcome to Minecraft Bedrock Server Manager for Windows.

Syntax:
	help
		Print this help message.
	@ COMMAND
		Send commands minecraft server directly.
	$ COMMAND
		Execute the shell command directly and print output.
	status
		Status of the bedrock server
		alias: s
	start
		Start the bedrock server
	stop
		Stop the bedrock server
	quit
		Exit the server manager shell. If server is running, will be stopped.
		alias: q, exit
	backup save BACKUP DESCRIPTION
		Save the current state as a backup.
		Example: backup save Built a gold farm
		alias: bs
	backup restore BACKUP_NAME 
		Restore the backup to the specified BACKUP. Use 'backup list' to get list.
		alias: br
	backup list [FILTER ...]
		List the available backups.
		Example: 'backup list saves/manual/* saves/periodic/20211002-*' will print all manual backups and periodic backup taken on 10/02 or.
		alias: bl
	workspace status
		Get current status of the files.
		alias: ws
	backup period INTERVAL
		Set automatic backup perid. Set to 0 to disable. If set, new timer is started.
		Example formats: 1h - 1 hour, 20m - 20 minutes, 30s - 30 seconds
		alias: bp
	backup delete BACKUP_NAME
		Delete the specified backup. You can specify wildcard as well.
		Example: backup delete saves/manual/202102* will delete all backups starting saves/manual/202102*.
		alias: bd
	backup prune CUTOFF_TIME INTERVAL
		Cleanup periodic backups older than CUTOFF_TIME. Keep the backups for every INTERVAL.
		For example, 'backup prune 3d 8h' will cleanup backups older than 3days. It will leave
		one backup every 8hours. The backup that is retained is chosen such that each backup is spaced at 8h.
		Example: 
			backup prune 3d 1d - Prune all backups older than 3days keeping one backup per day.
			In other words, turn into daily backup for backups older than 3d. Backups within last 3days will
			not be touched.
		Warning: Once deleted, backups cannot be restored through BedrockServerManager. You can
			salvage git commits through git.
	workspace clean
		Restore the current state to currently active backup. This deletes the modified files (since last backup).
		Current contents are backed as 'saved/temp/DATE_TIME'.
		alias: wc
		
	
	backup clean
`)
	return nil
}
