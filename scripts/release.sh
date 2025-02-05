#!/bin/bash

set -e  # Exit immediately if a command exits with a non-zero status

release_tag=$(echo "$1" | xargs)  # Trim whitespace
sha=$(echo "$2" | xargs)  # Trim whitespace

# Ensure a release tag is provided
if [[ -z "$release_tag" ]]; then
    echo "Usage: $0 vX.Y.Z <sha>"
    exit 1
fi

# Ensure the tag follows vX.Y.Z format
if ! [[ $release_tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Version must be in format vX.Y.Z"
    exit 1
fi

# Ensure a SHA is provided
if [[ -z "$sha" ]]; then
    echo "Error: SHA must be provided."
    exit 1
fi

# Verify version in conf/version.go matches with version
if ! grep -qc "var VERSION = \"$release_tag\"" conf/version.go; then
    echo "Error: Version mismatch. Update conf/version.go to $release_tag via PR first"
    exit 1
fi

# Ensure we are on the main branch
current_branch=$(git rev-parse --abbrev-ref HEAD)
if [[ "$current_branch" != "main" ]]; then
    echo "Error: Must switch to main branch"
    exit 1
fi

# Ensure there are no local changes
if ! git diff-index --quiet HEAD --; then
    echo "Error: You have uncommitted changes. Please commit or stash them before proceeding."
    exit 1
fi

# Check if the tag already exists
if git rev-parse "$release_tag" >/dev/null 2>&1; then
    echo "Error: Tag $release_tag already exists in repo. Please use a new version number."
    exit 1
fi

# Ensure the provided SHA exists in the repository
if ! git rev-parse --verify "$sha" >/dev/null 2>&1; then
    echo "Error: Provided commit SHA does not exist in the repository."
    exit 1
fi

# Confirm the release
read -p "All Pre-checks passed. Release version $release_tag at commit $sha? (y/N) " confirm
if [[ "$confirm" != "y" ]]; then
    echo "Release aborted."
    exit 1
fi

# Create and push the tag
git tag -a "$release_tag" "$sha" -m "Release $release_tag on $sha"
if ! git push origin "$release_tag"; then
    echo "Error: Failed to push tag. Deleting local tag."
    git tag -d "$release_tag"
    exit 1
fi

echo "Release $release_tag created on $sha and pushed successfully. Check https://github.com/heroku/heroku-integration-service-mesh/actions/workflows/release.yml for release status."