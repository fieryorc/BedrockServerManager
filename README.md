# BedrockServerManager
Minecraft Bedrock server manager for Windows

This is a simple Minecraft Bedrock server manager. Using this program,
you can manage your bedrock server and the world.

This program is released under MIT licence.

## Features
 * Interactive command line interface
 * Timestamped bedrock server logs
 * Manage the world backups using GIT. So backups are incremental and take less space.
 * Manual live backup
 * Backup restore (requires server to be stopped)
 * Automatic periodic live backups

## Why?
I run a small Bedrock server for myself and few friends. I run it on windows.
I was looking for a server manager for windows where it will take periodic live backups.
Most server managers copy files through zip which is a hassle to manage. I wanted something
that will use git. So I decided to write one on my own.

So here we are.

## Setting up
### Step 1: Install git
Download and install git from [git site](https://git-scm.com/download/win).

#### Set up PATH variables
If you set up git with default settings, git should be in your `PATH`. You can verify this by
opening a command prompt and type `git`. If this says not found, then you will need to add git to the
`PATH` following these instructions.

* Right click on the windows button, click System.
* In the system properties window, click `Advanced System Settings` -> `Environment Variables`.
* Edit the `PATH` variable (Either user or System will work) to include 
* git.exe directory (usually `C:\Program Files\Git\cmd`).

### Step 2: Install bedrock server
Download [bedrock server](https://www.minecraft.net/en-us/download/server/bedrock) and unzip to a directory of your choice.

### Step 3: Copy BedrockServerManager.exe to your bedrock directory

Download and copy the latest BedrockServerManager.exe to your Bedrock server directory (directory that contains `bedrock_server.exe`) you just created.

You can find lates release of BedrockServerManager under 
[releases section in github](https://github.com/fieryorc/BedrockServerManager/releases).

### STEP 4: Initialize git
 Open a command window, and navigate to your bedrock directory.

Create a new file named .gitignore and add the following to the file. These files will be ignored by the backup.
```
*.dll
*.exe
*.html
*.pdb
*.txt
valid_known_packs.json.json
behavior_packs
definitions
internalStorage
resource_packs
structures
```

```sh
git init .
git add .gitignore worlds\ permissions.json server.properties whitelist.json
git commit -m "Initial commit"
```

Verify if all is set by running `git status`. You should see something like:
```
On branch master
nothing to commit, working tree clean
```

Now proceed to next step.

### Step 4: Run the server manager

You can now run the program by typing `BedrockServerManager`. This will show you the
interactive prompt. By default the program is set up to back up every 30 minutes.


## Command Line options
You can run `BedrockServerManager -help` to get list of supported options.

## Troubleshooting
If you run into issues related to backup, exit the manager, run `git status` and make sure that
the directory is clean. Once you get the directory to clean state, backup issues should disappear.

## FAQs
Q: How to turn loggin on?

A: Verbose logs are written to %temp% directory. Look for recently modified files with name starting
with `BedrockServerManager.exe.XXX` or run `dir /OD %temp%\BedrockServerManager.exe.*` to get the log log file list. Alternatively, you can also pass --logtostderr flag to print more verbose logging to the console though it can be very distracting.

## Issues
Hope you find this useful and like it. If you find any issues, please report or send PR.

