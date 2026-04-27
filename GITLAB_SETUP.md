# GitLab Setup Guide

Your OpenCode Goal Tracker is ready to push to GitLab!

## 📍 Current Status

✅ All files committed to local git repository  
✅ GitLab CI/CD pipeline configured  
✅ Installation script updated for GitLab  
✅ Documentation updated  

## 🚀 Next Steps

### 1. Create GitLab Repository

Go to GitLab and create a new repository:
- **Namespace**: `sig`
- **Project name**: `opengoal`
- **Visibility**: Public or Private (your choice)
- **Initialize**: Leave unchecked (you already have files)

Or use the GitLab CLI:
```bash
# If you have glab installed
glab repo create sig/opengoal --public
```

### 2. Add GitLab Remote (if not already added)

```bash
cd /Users/VACDC6/dev/sig/opengoal

# Check current remote
git remote -v

# If origin is not set or wrong, update it
git remote add origin https://gitlab.com/sig/opengoal.git

# Or if origin exists, update it
git remote set-url origin https://gitlab.com/sig/opengoal.git
```

### 3. Push to GitLab

```bash
cd /Users/VACDC6/dev/sig/opengoal

# Push main branch
git push -u origin main
```

### 4. Create First Release

After pushing, create a release tag:

```bash
cd /Users/VACDC6/dev/sig/opengoal

# Create annotated tag
git tag -a v1.0.0 -m "Release v1.0.0 - Initial release"

# Push the tag
git push origin v1.0.0
```

This will trigger the GitLab CI/CD pipeline which will:
1. Build binaries for all platforms
2. Run tests
3. Create a GitLab Release
4. Attach all binaries to the release

## 📦 What Gets Built

The CI/CD pipeline builds:
- **macOS Intel** (x86_64)
- **macOS Apple Silicon** (ARM64)
- **Linux x86_64**
- **Linux ARM64**
- **Windows x86_64**

Plus SHA256 checksums for all binaries.

## 🔍 Monitor Pipeline

After pushing:

1. Go to: https://gitlab.com/sig/opengoal/-/pipelines
2. Watch the pipeline run (build → test → release)
3. Check for any errors

## 📥 Installation

Once released, users install with:

```bash
curl -fsSL https://gitlab.com/sig/opengoal/-/raw/main/install.sh | bash
```

## 🎯 Testing Installation

Test on a different machine or clean environment:

```bash
# Install
curl -fsSL https://gitlab.com/sig/opengoal/-/raw/main/install.sh | bash

# Restart OpenCode or reload config

# Test commands
/today
/goals-main Test goal
/task-add Test task
```

## 📊 Release Checklist

Before each release:

- [ ] Update version in `.gitlab-ci.yml`
- [ ] Update version in `install.sh`
- [ ] Update CHANGELOG.md
- [ ] Test all commands locally
- [ ] Commit changes
- [ ] Create and push tag
- [ ] Verify pipeline completes
- [ ] Test installation from GitLab
- [ ] Update README if needed

## 🔐 Authentication

If you need to authenticate:

```bash
# GitLab Personal Access Token
# Go to: GitLab → Settings → Access Tokens
# Create token with: api, read_repository, write_repository

# Set git credential helper
git config --global credential.helper store

# Or use SSH keys (recommended)
# Add your SSH key at: GitLab → Settings → SSH Keys
git remote set-url origin git@gitlab.com:sig/opengoal.git
```

## 📁 Repository Structure

```
gitlab.com/sig/opengoal/
├── main.go                  # Application source
├── go.mod                   # Go module
├── .gitlab-ci.yml          # CI/CD pipeline ⭐
├── install.sh              # Installation script
├── build.sh                # Build script
├── README.md               # Main documentation
├── CHANGELOG.md            # Version history
├── LICENSE                 # MIT license
├── commands/               # OpenCode commands
│   ├── goals-*.md
│   ├── task-*.md
│   └── today.md
└── docs/
    ├── GETTING_STARTED.md
    ├── DISTRIBUTION.md
    └── ...
```

## 🎉 Current Commit

```
commit 370a505
Author: VACDC6
Date:   Thu Apr 23 18:XX:XX 2026

    Initial release v1.0.0
    
    - Lightning-fast goal and task tracking for OpenCode
    - Written in Go for maximum performance (12ms response time)
    - Main goals with sub-goals and automatic progress tracking
    - Task list with priority levels
    - Today dashboard showing complete overview
    - Persistent memory across sessions
    - Cross-platform support (macOS, Linux, Windows)
    - GitLab CI/CD pipeline for automated releases
    - Comprehensive documentation and installation scripts
```

## 🛠️ Troubleshooting

### Pipeline Fails

Check:
- Go version in `.gitlab-ci.yml`
- Syntax in YAML file
- Build commands work locally

### Can't Push

Check:
- Remote URL is correct
- You have access to the repository
- Authentication is set up

### Installation Fails

Check:
- Install script URL is correct
- Binaries are attached to release
- Permissions are correct

## 🎊 You're Ready!

Everything is committed and ready to push. Just run:

```bash
cd /Users/VACDC6/dev/sig/opengoal
git push -u origin main
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

Then share the installation link:
```bash
curl -fsSL https://gitlab.com/sig/opengoal/-/raw/main/install.sh | bash
```

Good luck! 🚀
