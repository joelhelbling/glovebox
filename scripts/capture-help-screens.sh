#!/bin/bash
#
# Capture all help screens from the glovebox CLI
# Outputs to docs/help_screens.md
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
OUTPUT_FILE="$PROJECT_ROOT/docs/help_screens.md"
GB="$PROJECT_ROOT/bin/glovebox"

# Ensure the binary exists
if [[ ! -x "$GB" ]]; then
    echo "Error: glovebox binary not found at $GB" >&2
    echo "Run 'make build' first" >&2
    exit 1
fi

# Ensure docs directory exists
mkdir -p "$PROJECT_ROOT/docs"

# Helper function to add a command's help to the output
add_help() {
    local title="$1"
    local cmd="$2"

    echo "## $title"
    echo "Command: \`gb ${cmd}\`"
    echo ""
    echo '```'
    $GB $cmd 2>&1 || true
    echo '```'
    echo ""
}

# Generate the markdown file
{
    echo "# Help Screens"
    echo ""
    echo "All help output from the \`glovebox\` app."
    echo ""

    # Root command
    add_help "Root" "--help"

    # Top-level commands (alphabetical)
    add_help "Add" "add --help"
    add_help "Build" "build --help"
    add_help "Clean" "clean --help"
    add_help "Clone" "clone --help"
    add_help "Completion" "completion --help"
    add_help "Completion: Bash" "completion bash --help"
    add_help "Completion: Fish" "completion fish --help"
    add_help "Completion: PowerShell" "completion powershell --help"
    add_help "Completion: Zsh" "completion zsh --help"
    add_help "Help" "help --help"
    add_help "Init" "init --help"
    add_help "Mod" "mod --help"
    add_help "Mod: Cat" "mod cat --help"
    add_help "Mod: Create" "mod create --help"
    add_help "Mod: List" "mod list --help"
    add_help "Remove" "remove --help"
    add_help "Run" "run --help"
    add_help "Status" "status --help"

} > "$OUTPUT_FILE"

echo "Help screens captured to $OUTPUT_FILE"
