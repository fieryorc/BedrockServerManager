# BedrockServerManager
Minecraft Bedrock server manager for Windows

This is a simple Minecraft Bedrock server manager. Using this program,
you can manage your bedrock server and the world.

This program is released under MIT licence.

## Features
 * Start/Stop bedrock server
 * Manage the world backups using GIT
 * Interactive command line
 * Manual backup/restore
 * Automatic periodic backups

## Setting up
### Step 1: Install git
Download and install git from [git site](https://git-scm.com/download/win)

#### Set up PATH variables
If you set up git with default settings, git should be in your `PATH`. You can verify this by
opening a command prompt and type `git`. If this says not found, then you will need to add git to the
`PATH` following these instructions.

* Right click on the windows button, click System.
* In the system properties window, click `Advanced System Settings` -> `Environment Variables`.
* Edit the `PATH` variable (Either user or System will work) to include 
* git.exe directory (usually `C:\Program Files\Git\cmd`).

### Step 2: Copy BedrockServerManager.exe

Download and copy the latest BedrockServerManager.exe to your Bedrock server directory (directory that contains `bedrock_server.exe`).
You can find it under releases section.

### STEP 3: Initialize git
 Open a command window, and navigate to your bedrock directory.

Create a new file named .gitignore and add the following to the file.
```
*.dll
*.exe
*.html
*.json
*.pdb
*.txt
behavior_packs
definitions
internalStorage
resource_packs
structures
```

```sh
cd BEDROCK_DIRECTORY
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
