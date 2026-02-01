### S1Resolver

S1Resolver is a CLI tool designed for SOC analysts to quickly resolve open threats and alerts in SentinelOne consoles without the need to manually authenticate.

#### Installation Location
By default, the installer creates a visible folder in your home directory to store the binary and configuration: ~/S1Resolver

It also automatically registers the command alias `s1` in your terminal.

##### Customizing the Install Location:
If you prefer to install the tool elsewhere, edit the install_s1resolver.py script before running it.

Change Line 9:
`INSTALL_DIR = os.path.expanduser("~/S1Resolver")`
*Update the path inside the quotes to your preferred location.*

OR If you have already installed the tool and want to move it to a new location:

Move the folder:
Ex: `mv ~/S1Resolver ~/scripts/S1Resolver`

Then open your zsh configuration: `nano ~/.zshrc`
And find the S1Resolver section near the bottom.

**IMPORTANT - Update the export PATH line to match your new location.**

Apply Changes: `source ~/.zshrc`

#### Updating API Keys
To add a new console token or update an expired one:

Navigate to the installation folder.

Open config.json in any text editor.

Paste your new token into the "token" field for the relevant console.

Save the file.
