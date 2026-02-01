#!/usr/bin/env python3
import os
import json
import shutil
import sys
import subprocess

# --- CONFIGURATION ---
INSTALL_DIR = os.path.expanduser("~/S1Resolver")
BINARY_NAME = "S1Resolver"
CONFIG_NAME = "config.json"
ZSHRC_PATH = os.path.expanduser("~/.zshrc")

# Zsh Configuration Block
ZSH_BLOCK = f"""
# --- S1Resolver Tool ---
# Added by install_s1resolver.py
export PATH="{INSTALL_DIR}:$PATH"

# Create shortcut 's1' so you don't have to type the full name
alias s1="{BINARY_NAME}"

# Enable Autocomplete for both S1Resolver AND s1
autoload -Uz compinit && compinit
_s1resolver_completion() {{
    local -a consoles
    consoles=("${{(@f)$({BINARY_NAME} -list-consoles)}}")
    _arguments \\
        '-c[Select Console]:console:($consoles)' \\
        '-d[Select Default Site]' \\
        '-t[Threats Mode]' \\
        '-a[Alerts Mode]'
}}
# Register completion for both commands
compdef _s1resolver_completion {BINARY_NAME} s1
"""

def print_color(text, color_code):
    print(f"\033[{color_code}m{text}\033[0m")

def main():
    print_color("\n=== S1Resolver Installer ===", "1;36") # Cyan
    
    # 1. Verify Binary exists
    current_binary = os.path.join(os.getcwd(), BINARY_NAME)
    if not os.path.exists(current_binary):
        print_color(f"Error: Could not find '{BINARY_NAME}' file in this folder.", "91")
        print(f"Please make sure the '{BINARY_NAME}' binary is in the same folder as this script.")
        return

    # 2. Configure Consoles (Interactive Mode)
    print("\nLet's configure your SentinelOne consoles.")
    print("You will need the Console Name, URL, and API Token for each environment.")
    print("Press Enter on an empty 'Console Name' to finish.")
    
    user_config = []
    
    while True:
        print("\n--- Add New Console ---")
        name = input("Console Name (e.g., NFR, Prod): ").strip()
        if not name:
            break
            
        url = input(f"URL for '{name}' (e.g., https://usea1-001.sentinelone.net): ").strip()
        if url and not url.startswith("http"):
             print_color("Warning: URL should usually start with https://", "33")
             
        token = input(f"API Token for '{name}': ").strip()
        
        if name and url and token:
            user_config.append({
                "name": name,
                "url": url,
                "token": token
            })
            print_color(f"Added '{name}'!", "32")
        else:
            print_color("Skipping... (Missing fields)", "33")

    if not user_config:
        print_color("\nNo configuration created. (Proceeding with empty config...)", "33")

    # 3. Create Install Directory
    if not os.path.exists(INSTALL_DIR):
        print(f"\nCreating directory: {INSTALL_DIR}")
        os.makedirs(INSTALL_DIR)

    # 4. Move Binary
    dest_binary = os.path.join(INSTALL_DIR, BINARY_NAME)
    try:
        if os.path.exists(dest_binary):
            os.remove(dest_binary) # Overwrite existing
        shutil.copy2(current_binary, dest_binary)
        os.chmod(dest_binary, 0o755)
        print(f"Installed '{BINARY_NAME}' to {INSTALL_DIR}")
    except Exception as e:
        print_color(f"Error moving binary: {e}", "91")
        return

    # 5. Write Config JSON
    config_path = os.path.join(INSTALL_DIR, CONFIG_NAME)
    try:
        if not user_config and os.path.exists(config_path):
             print("No new inputs provided. Keeping existing config.json.")
        else:
            with open(config_path, 'w') as f:
                json.dump(user_config, f, indent=4)
            print(f"Configuration saved to {config_path}")
    except Exception as e:
        print_color(f"Error saving config: {e}", "91")
        return

    # 6. Update Zshrc
    already_installed = False
    if os.path.exists(ZSHRC_PATH):
        with open(ZSHRC_PATH, 'r') as f:
            if "# --- S1Resolver Tool ---" in f.read():
                already_installed = True

    if not already_installed:
        print("Adding configuration to ~/.zshrc...")
        with open(ZSHRC_PATH, 'a') as f:
            f.write(ZSH_BLOCK)
    else:
        print("Zsh configuration already exists. Skipping...")

    # 7. Finish
    print_color("\nSuccess! Installation Complete.", "1;32") # Green
    print("Please restart your terminal or run:")
    print_color(f"  source {ZSHRC_PATH}", "1;33") # Yellow
    print(f"\nThen try typing 's1' to start!")

if __name__ == "__main__":
    main()
