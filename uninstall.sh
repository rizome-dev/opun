#!/usr/bin/env bash

set -euo pipefail

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Print colored output
print_error() { echo -e "${RED}✗ $1${NC}" >&2; }
print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_info() { echo -e "$1"; }

# Check if running as root
check_root() {
    if [[ $EUID -eq 0 ]]; then
        print_error "This script should not be run as root"
        exit 1
    fi
}

# Detect if installed via Homebrew
check_brew_installation() {
    if command -v brew >/dev/null 2>&1; then
        if brew list opun >/dev/null 2>&1; then
            print_warning "Opun appears to be installed via Homebrew"
            print_info "Please run: brew uninstall opun"
            print_info ""
            read -p "Continue with manual uninstall anyway? (y/N) " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                exit 0
            fi
        fi
    fi
}

# Remove binary
remove_binary() {
    local binary_paths=(
        "/usr/local/bin/opun"
        "$HOME/.local/bin/opun"
        "$HOME/bin/opun"
    )
    
    local removed=false
    for path in "${binary_paths[@]}"; do
        if [[ -f "$path" ]]; then
            print_info "Removing binary: $path"
            if rm -f "$path" 2>/dev/null || sudo rm -f "$path"; then
                print_success "Removed binary: $path"
                removed=true
            else
                print_error "Failed to remove binary: $path"
            fi
        fi
    done
    
    if [[ "$removed" == false ]]; then
        print_warning "No opun binary found in standard locations"
    fi
}

# Remove configuration directory
remove_config() {
    local config_dir="$HOME/.opun"
    
    if [[ -d "$config_dir" ]]; then
        print_info "Found configuration directory: $config_dir"
        
        # Check if directory has content
        if [[ -n "$(ls -A "$config_dir" 2>/dev/null)" ]]; then
            print_warning "Configuration directory contains:"
            ls -la "$config_dir" | head -10
            if [[ $(ls -1 "$config_dir" | wc -l) -gt 9 ]]; then
                print_info "... and $(( $(ls -1 "$config_dir" | wc -l) - 9 )) more items"
            fi
            echo
            read -p "Remove configuration directory and all its contents? (y/N) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                rm -rf "$config_dir"
                print_success "Removed configuration directory"
            else
                print_info "Keeping configuration directory"
            fi
        else
            rm -rf "$config_dir"
            print_success "Removed empty configuration directory"
        fi
    else
        print_info "No configuration directory found"
    fi
}

# Remove shell completions
remove_completions() {
    local completion_files=(
        # Bash completions
        "/usr/local/etc/bash_completion.d/opun"
        "/etc/bash_completion.d/opun"
        "$HOME/.local/share/bash-completion/completions/opun"
        
        # Zsh completions
        "/usr/local/share/zsh/site-functions/_opun"
        "/usr/share/zsh/site-functions/_opun"
        "$HOME/.zsh/completions/_opun"
        
        # Fish completions
        "/usr/local/share/fish/vendor_completions.d/opun.fish"
        "/usr/share/fish/vendor_completions.d/opun.fish"
        "$HOME/.config/fish/completions/opun.fish"
    )
    
    local removed=false
    for file in "${completion_files[@]}"; do
        if [[ -f "$file" ]]; then
            if rm -f "$file" 2>/dev/null || sudo rm -f "$file"; then
                print_success "Removed completion file: $file"
                removed=true
            else
                print_error "Failed to remove completion file: $file"
            fi
        fi
    done
    
    if [[ "$removed" == false ]]; then
        print_info "No shell completion files found"
    fi
}

# Remove from PATH if added to shell configs
check_path_modifications() {
    local shell_configs=(
        "$HOME/.bashrc"
        "$HOME/.bash_profile"
        "$HOME/.zshrc"
        "$HOME/.profile"
    )
    
    local found=false
    for config in "${shell_configs[@]}"; do
        if [[ -f "$config" ]] && grep -q "opun" "$config" 2>/dev/null; then
            print_warning "Found 'opun' references in $config"
            found=true
        fi
    done
    
    if [[ "$found" == true ]]; then
        print_info "Please manually check and remove any opun-related PATH modifications from your shell configuration files"
    fi
}

# Main uninstall function
main() {
    echo "Opun Uninstaller"
    echo "================"
    echo
    
    check_root
    check_brew_installation
    
    print_info "This will remove:"
    print_info "  • Opun binary"
    print_info "  • Configuration directory (~/.opun)"
    print_info "  • Shell completion files"
    echo
    
    read -p "Continue with uninstall? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Uninstall cancelled"
        exit 0
    fi
    
    echo
    remove_binary
    remove_config
    remove_completions
    check_path_modifications
    
    echo
    print_success "Opun has been uninstalled"
    
    # Check if opun is still in PATH
    if command -v opun >/dev/null 2>&1; then
        print_warning "opun command is still available in your PATH"
        print_info "Location: $(command -v opun)"
        print_info "You may need to restart your shell or manually remove this file"
    fi
}

# Run main function
main "$@"