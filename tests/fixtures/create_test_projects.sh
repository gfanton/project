#!/usr/bin/env bash
# Create sample test projects for testing

set -euo pipefail

# Source test helpers
source "$(dirname "${BASH_SOURCE[0]}")/../helpers/test_helpers.sh"

create_sample_projects() {
    local project_root="${1:-$TEST_PROJECT_DIR}"
    
    log_info "Creating sample projects in $project_root"
    
    # Project 1: gfanton/project
    create_project "$project_root" "gfanton" "project" "Main project management tool"
    
    # Project 2: gfanton/dotfiles  
    create_project "$project_root" "gfanton" "dotfiles" "Personal configuration files"
    
    # Project 3: testorg/webapp
    create_project "$project_root" "testorg" "webapp" "Sample web application"
    
    # Project 4: testorg/api
    create_project "$project_root" "testorg" "api" "REST API server"
    
    # Project 5: opensource/library
    create_project "$project_root" "opensource" "library" "Open source library"
    
    log_success "Sample projects created successfully"
}

create_project() {
    local root="$1"
    local org="$2"
    local name="$3"
    local description="$4"
    
    local project_dir="$root/$org/$name"
    
    log_info "Creating project: $org/$name"
    mkdir -p "$project_dir"
    cd "$project_dir"
    
    # Initialize git
    git init --quiet
    git config user.name "Test User"
    git config user.email "test@example.com"
    
    # Create project structure
    echo "# $name" > README.md
    echo "" >> README.md
    echo "$description" >> README.md
    echo "" >> README.md
    echo "This is a test project for tmux integration testing." >> README.md
    
    # Create some sample files
    mkdir -p src docs
    echo "console.log('Hello from $name');" > src/main.js
    echo "# Documentation for $name" > docs/README.md
    
    # Create .gitignore
    cat > .gitignore << EOF
node_modules/
build/
*.log
.DS_Store
EOF
    
    # Initial commit
    git add .
    git commit --quiet -m "Initial commit: $description"
    
    # Create additional branches for testing
    git checkout -b feature/new-feature --quiet
    echo "// New feature implementation" >> src/main.js
    git add src/main.js
    git commit --quiet -m "Add new feature"
    
    git checkout -b bugfix/fix-issue --quiet
    echo "# Bug fixes" >> README.md
    git add README.md
    git commit --quiet -m "Fix important bug"
    
    # Back to main
    git checkout main --quiet
    
    log_success "Project created: $project_dir"
}

# If run directly, create sample projects
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ -z "${TEST_PROJECT_DIR:-}" ]]; then
        echo "ERROR: TEST_PROJECT_DIR not set. Run from test environment." >&2
        exit 1
    fi
    
    create_sample_projects "$TEST_PROJECT_DIR"
fi