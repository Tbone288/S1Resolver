
S1Resolver is a high-performance CLI tool designed for SOC analysts to instantly identify and resolve threats and alerts across multiple SentinelOne consoles.

Installation Location
By default, the installer creates a visible folder in your home directory to store the binary and configuration: ~/S1Resolver

It also automatically registers the command alias s1 in your terminal.

Customizing the Install Location:
If you prefer to install the tool elsewhere (e.g., inside a tools folder), edit the install_s1resolver.py script before running it.

Change Line 9:

Python
INSTALL_DIR = os.path.expanduser("~/S1Resolver")
Update the path inside the quotes to your preferred location.

Moving the Tool
If you have already installed the tool and want to move it to a new location:

Move the folder:

mv ~/S1Resolver ~/scripts/S1Resolver
Update your Shell Config:

Open your zsh configuration: nano ~/.zshrc

Find the S1Resolver section near the bottom.

Update the export PATH line to match your new location.

Apply Changes:

Run: source ~/.zshrc

Configuration & API Keys
To add a new console token or update an expired one:

Navigate to the installation folder.

Open config.json in any text editor.

Paste your new token into the "token" field for the relevant console.

Save the file. The changes are instant.
