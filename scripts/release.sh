#!/bin/bash
set -euo pipefail

# release.sh - Helper script for managing stackaroo releases
# Usage: ./scripts/release.sh [version]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colours for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Colour

# Logging functions
log_info() {
    echo -e "${BLUE}$1${NC}"
}

log_success() {
    echo -e "${GREEN}$1${NC}"
}

log_warning() {
    echo -e "${YELLOW}$1${NC}"
}

log_error() {
    echo -e "${RED}$1${NC}"
    exit 1
}

# Check if we're in the project root
check_project_root() {
    if [[ ! -f "${PROJECT_ROOT}/go.mod" ]] || [[ ! -f "${PROJECT_ROOT}/VERSION" ]]; then
        log_error "This script must be run from the stackaroo project root directory"
    fi
}

# Validate version format (semantic versioning)
validate_version() {
    local version="$1"

    # Remove 'v' prefix if present
    version="${version#v}"

    # Check semantic version format (X.Y.Z with optional pre-release suffix)
    if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+(\.[a-zA-Z0-9]+)*)?$ ]]; then
        log_error "Invalid version format: $version. Expected format: 1.2.3 or 1.2.3-rc1"
    fi

    echo "$version"
}

# Check if git working directory is clean
check_git_clean() {
    if [[ -n "$(git status --porcelain)" ]]; then
        log_error "Git working directory is not clean. Please commit or stash changes before releasing."
    fi
    log_success "Git working directory is clean"
}

# Check if we're on the main/master branch
check_main_branch() {
    local current_branch
    current_branch="$(git rev-parse --abbrev-ref HEAD)"

    if [[ "$current_branch" != "main" && "$current_branch" != "master" ]]; then
        log_warning "Not on main/master branch (currently on: $current_branch)"
        read -p "Continue anyway? [y/N]: " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_error "Aborted by user"
        fi
    fi
    log_success "Branch check passed"
}

# Run pre-release checks
run_pre_release_checks() {
    log_info "Running pre-release checks..."

    # Run tests
    log_info "Running tests..."
    if ! make test >/dev/null 2>&1; then
        log_error "Tests failed. Please fix before releasing."
    fi
    log_success "Tests passed"

    # Run linting
    log_info "Running linting..."
    if ! make lint >/dev/null 2>&1; then
        log_error "Linting failed. Please fix before releasing."
    fi
    log_success "Linting passed"

    # Check for security vulnerabilities
    log_info "Running security scan..."
    if ! go install golang.org/x/vuln/cmd/govulncheck@latest >/dev/null 2>&1; then
        log_warning "Could not install govulncheck"
    elif ! govulncheck ./... >/dev/null 2>&1; then
        log_error "Security vulnerabilities found. Please fix before releasing."
    else
        log_success "Security scan passed"
    fi

    # Verify dependencies
    log_info "Verifying dependencies..."
    if ! go mod verify >/dev/null 2>&1; then
        log_error "Dependency verification failed"
    fi
    log_success "Dependencies verified"
}

# Update VERSION file
update_version_file() {
    local version="$1"
    echo "$version" > "${PROJECT_ROOT}/VERSION"
    log_success "Updated VERSION file to $version"
}

# Create and push git tag
create_git_tag() {
    local version="$1"
    local tag="v$version"

    # Check if tag already exists
    if git rev-parse "$tag" >/dev/null 2>&1; then
        log_error "Tag $tag already exists"
    fi

    # Stage VERSION file
    git add VERSION

    # Commit version update
    git commit -m "release: $tag"
    log_success "Committed version update"

    # Create annotated tag
    git tag -a "$tag" -m "Release $tag"
    log_success "Created tag $tag"

    # Push to origin
    log_info "Pushing to origin..."
    git push origin HEAD
    git push origin "$tag"
    log_success "Pushed tag $tag to origin"
}

# Show usage information
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS] <version>

Helper script for creating stackaroo releases.

Arguments:
    <version>       Version to release (e.g., 1.2.3 or v1.2.3)

Options:
    -h, --help     Show this help message
    --dry-run      Perform all checks but don't create the tag
    --skip-checks  Skip pre-release checks (tests, lint, etc.)

Examples:
    $0 1.2.3           # Release version 1.2.3
    $0 v1.2.4-rc1      # Release version 1.2.4-rc1
    $0 --dry-run 1.2.5 # Validate everything for 1.2.5 but don't tag

This script will:
1. Validate the version format
2. Check git working directory is clean
3. Run tests, linting, and security scans
4. Update the VERSION file
5. Create and push a git tag

The GitHub Actions workflow will automatically trigger when the tag is pushed.
EOF
}

# Main function
main() {
    local version=""
    local dry_run=false
    local skip_checks=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            --dry-run)
                dry_run=true
                shift
                ;;
            --skip-checks)
                skip_checks=true
                shift
                ;;
            -*)
                log_error "Unknown option: $1"
                ;;
            *)
                if [[ -z "$version" ]]; then
                    version="$1"
                else
                    log_error "Too many arguments. Expected one version argument."
                fi
                shift
                ;;
        esac
    done

    # Check if version was provided
    if [[ -z "$version" ]]; then
        log_error "Version argument is required. Use --help for usage information."
    fi

    # Validate and normalise version
    version="$(validate_version "$version")"
    log_info "Preparing release for version: $version"

    # Check project setup
    check_project_root

    # Git checks
    check_git_clean
    check_main_branch

    # Pre-release checks
    if [[ "$skip_checks" == false ]]; then
        run_pre_release_checks
    else
        log_warning "Skipping pre-release checks as requested"
    fi

    # Update VERSION file
    update_version_file "$version"

    if [[ "$dry_run" == true ]]; then
        log_warning "Dry run mode - not creating git tag"
        log_info "Would create tag: v$version"

        # Restore original VERSION file
        git checkout -- VERSION
        log_info "Restored original VERSION file"
    else
        # Create and push tag
        create_git_tag "$version"

        log_success "Release v$version completed!"
        log_info "GitHub Actions will now build and publish the release."
        log_info "Monitor the release at: https://codeberg.org/orien/stackaroo/actions"
    fi
}

# Run main function with all arguments
main "$@"
