#!/usr/bin/env fish
# Find directories with __periodics.yaml files that are missing a specific release version
#
# Usage: find-missing-periodics.fish REPO_PATH RELEASE
# Example: find-missing-periodics.fish /path/to/repo 4.21

set repo_path $argv[1]
set release $argv[2]

if test -z "$repo_path" -o -z "$release"
    echo "Usage: find-missing-periodics.fish REPO_PATH RELEASE" >&2
    echo "Example: find-missing-periodics.fish /path/to/repo 4.21" >&2
    exit 1
end

if not test -d "$repo_path"
    echo "Error: Directory '$repo_path' does not exist" >&2
    exit 1
end

# Find all __periodics.yaml files and extract their directories
set -l all_dirs (find "$repo_path/ci-operator/config" -name '*__periodics.yaml' 2>/dev/null | xargs -r dirname | sort -u)

# For each directory, check if it has at least one periodic file but no file for the given release
for dir in $all_dirs
    set -l periodic_files (find "$dir" -maxdepth 1 -name '*__periodics.yaml')
    set -l release_files (find "$dir" -maxdepth 1 -name "*$release*__periodics.yaml")

    # If we have periodic files but no release-specific files, output the directory
    if test (count $periodic_files) -gt 0 -a (count $release_files) -eq 0
        # Output relative path from repo root
        set -l rel_dir (string replace "$repo_path/" "" "$dir")
        echo "$rel_dir"
        for file in $periodic_files
            set -l rel_file (string replace "$repo_path/" "" "$file")
            echo "  $rel_file"
        end
    end
end
