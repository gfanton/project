#!/usr/bin/env bash
set -e

echo "🚀 Setting up enhanced completion test environment..."

# Build the project first
make build

# Create a temporary test environment with more projects
TEST_DIR=$(mktemp -d)
echo "Test environment: $TEST_DIR"

# Create test project structure with many projects for better completion testing
mkdir -p "$TEST_DIR/code"
projects=(
    "user1/awesome-project"
    "user1/cool-app" 
    "user1/project-alpha"
    "user1/project-beta"
    "user1/project-gamma"
    "user2/amazing-tool"
    "user2/awesome-app"
    "user2/project-delta"
    "user2/super-project"
    "user3/fantastic-app"
    "user3/great-project"
    "user3/project-epsilon" 
    "user3/wonderful-tool"
    "myorg/backend-service"
    "myorg/frontend-app"
    "myorg/mobile-app"
    "myorg/project-zeta"
    "company/api-server"
    "company/web-client"
    "company/project-omega"
)

for project in "${projects[@]}"; do
    project_path="$TEST_DIR/code/$project"
    mkdir -p "$project_path"
    (cd "$project_path" && git init --quiet)
    echo "Created: $project"
done

# Create config file
cat > "$TEST_DIR/.projectrc" <<EOF
root = "$TEST_DIR/code"
user = "testuser"
EOF

# Generate project init script
./build/proj init zsh > "$TEST_DIR/project.zsh"

# Create enhanced zshrc with completion configuration
cat > "$TEST_DIR/.zshrc" <<EOF
# Enhanced zshrc for project completion testing
export PROJECT_CONFIG_FILE="$TEST_DIR/.projectrc"

# Enable completion system with menu selection
autoload -U compinit && compinit -u

# Configure zsh options for better completion experience
setopt AUTO_MENU           # Show completion menu on successive tab press
setopt AUTO_LIST           # Automatically list choices on ambiguous completion
setopt COMPLETE_IN_WORD    # Complete from both ends of a word
setopt ALWAYS_TO_END       # Move cursor to the end if word had one match

# Load project completion
source "$TEST_DIR/project.zsh"

# Welcome message
echo "🎯 Enhanced Project Completion Test Environment!"
echo ""
echo "📁 Available projects (${#projects[@]} total):"
printf "   %s\n" "${projects[@]}" | head -10
echo "   ... and $((${#projects[@]} - 10)) more!"
echo ""
echo "🔄 Completion Features:"
echo "   • Type 'p proj<TAB>' to see multiple project options"
echo "   • Use TAB to cycle through matches"  
echo "   • Use arrow keys to navigate in menu selection mode"
echo "   • Try: p awesome<TAB>, p project<TAB>, p user1/<TAB>"
echo ""
echo "💡 Tips:"
echo "   • TAB once: show options"
echo "   • TAB again: enter menu selection"
echo "   • Arrow keys: navigate menu"
echo "   • Enter: select option"
echo "   • Ctrl+C: cancel completion"
echo ""
EOF

# Add the projects array to the zshrc
cat >> "$TEST_DIR/.zshrc" <<EOF
# Projects array for reference
projects=($(printf '"%s" ' "${projects[@]}"))
EOF

# Export environment and start zsh
export ZDOTDIR="$TEST_DIR"
export PROJECT_CONFIG_FILE="$TEST_DIR/.projectrc"

echo "🎉 Starting zsh with enhanced completion..."
echo "💡 Try typing: p proj<TAB> to see multiple completions!"
echo "📝 Type 'exit' when done to cleanup."

# Start zsh
zsh

# Cleanup
echo "🧹 Cleaning up test environment..."
rm -rf "$TEST_DIR"
echo "✅ Done!"