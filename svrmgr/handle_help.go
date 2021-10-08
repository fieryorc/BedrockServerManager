package svrmgr

import (
	"context"
	"fmt"
)

type helpHandler struct{}

func initHelpHandler(provider Provider) {
	Register("help", &helpHandler{})
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
	exit
		Exit the server manager shell. If server is running, will be stopped.
		alias: e
	backup save [Optional backup message]
		Take a backup. Specify optional message. 
		alias: bf
	backup restore HASH_ID 
		Restore the backup to the specified HASH_ID. To get list of backups and hash ids, use backup list.
		alias: br
	backup list [RECENT_MAX_ITEMS]
		List backups specifying number of recent items to list.
		Example: 'backup list 20' will print last 20 items.
		alias: bl
	backup period INTERVAL
		Set automatic backup perid. Set to 0 to disable. If set, new timer is started.
		Example formats: 1h - 1 hour, 20m - 20 minutes, 30s - 30 seconds
		alias: bp
`)
	return nil
}
