# Distribution Guide

How to share OpenCode Goal Tracker with others.

## 📦 Distribution Options

### 1. GitHub Repository (Recommended)

**Best for**: Open source distribution, version control, community contributions

**Steps:**
1. Create a new GitHub repository
2. Copy all files from `~/.config/opencode/skills/goal-tracker/`
3. Push to GitHub
4. Create releases with binaries

**Advantages:**
- Automatic releases via GitHub Actions
- Version control
- Issue tracking
- Community contributions
- Free hosting

**Setup:**
```bash
cd ~/.config/opencode/skills/goal-tracker
git init
git add .
git commit -m "Initial commit"
git remote add origin https://github.com/yourusername/opencode-goal-tracker.git
git push -u origin main
```

### 2. Pre-compiled Binaries

**Best for**: Users who just want to install quickly

**Create binaries:**
```bash
cd ~/.config/opencode/skills/goal-tracker
./build.sh
```

This creates binaries in `dist/` for:
- macOS (Intel & Apple Silicon)
- Linux (x86_64 & ARM64)
- Windows (x86_64)

**Distribution methods:**
- GitHub Releases
- Your own website
- Package managers (Homebrew, apt, etc.)

### 3. Go Package

**Best for**: Go developers who want to build from source

Users can install with:
```bash
go install github.com/yourusername/opencode-goal-tracker@latest
```

### 4. OpenCode Skills Registry (Future)

When OpenCode adds a skills registry, you can publish there for one-command installation.

## 🚀 Recommended: GitHub + Releases

This is the best approach for maximum reach:

### Step-by-Step Guide

#### 1. Create GitHub Repository

1. Go to GitHub and create new repository: `opencode-goal-tracker`
2. Make it public (for community access)
3. Add description: "Lightning-fast goal tracker for OpenCode"
4. Add topics: `opencode`, `productivity`, `goals`, `golang`

#### 2. Prepare Your Repository

```bash
cd ~/.config/opencode/skills/goal-tracker

# Initialize git if not already done
git init

# Add files
git add .

# Commit
git commit -m "Initial release v1.0.0"

# Add remote
git remote add origin https://github.com/yourusername/opencode-goal-tracker.git

# Push
git branch -M main
git push -u origin main
```

#### 3. Create First Release

1. Build binaries:
```bash
./build.sh
```

2. Create a git tag:
```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

3. GitHub Actions will automatically:
   - Build binaries for all platforms
   - Create a release
   - Upload binaries
   - Generate checksums

#### 4. Update Install Script

Edit `install.sh` and replace:
```bash
REPO_URL="https://github.com/yourusername/opencode-goal-tracker"
```

With your actual GitHub URL.

#### 5. Test Installation

On a different machine (or clean environment):
```bash
curl -fsSL https://raw.githubusercontent.com/yourusername/opencode-goal-tracker/main/install.sh | bash
```

### Release Checklist

Before each release:

- [ ] Update version in `install.sh`
- [ ] Update version in `build.sh`
- [ ] Update CHANGELOG.md
- [ ] Test all commands locally
- [ ] Build and test all binaries
- [ ] Create git tag
- [ ] Verify GitHub Actions completes
- [ ] Test install script
- [ ] Update README if needed

## 📱 Alternative Distribution Methods

### Homebrew (macOS/Linux)

Create a Homebrew tap:

```ruby
# Formula/goals.rb
class Goals < Formula
  desc "Lightning-fast goal tracker for OpenCode"
  homepage "https://github.com/yourusername/opencode-goal-tracker"
  url "https://github.com/yourusername/opencode-goal-tracker/archive/v1.0.0.tar.gz"
  sha256 "..."
  
  depends_on "go" => :build
  
  def install
    system "go", "build", *std_go_args(ldflags: "-s -w")
  end
  
  test do
    system "#{bin}/goals", "list"
  end
end
```

Users install with:
```bash
brew tap yourusername/tap
brew install goals
```

### Docker

Create `Dockerfile`:
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o goals main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/goals /usr/local/bin/
ENTRYPOINT ["goals"]
```

Users run with:
```bash
docker run -v ~/.local/share/opencode:/data ghcr.io/yourusername/goals today
```

### NPM Package

Wrap the binary in an npm package for Node.js users.

Create `package.json`:
```json
{
  "name": "opencode-goal-tracker",
  "version": "1.0.0",
  "bin": {
    "goals": "./bin/goals"
  },
  "scripts": {
    "postinstall": "node scripts/install-binary.js"
  }
}
```

Users install with:
```bash
npm install -g opencode-goal-tracker
```

## 📢 Promotion

Once published, promote via:

1. **OpenCode Discord** - Share in #plugins or #tools channel
2. **Reddit** - r/golang, r/productivity
3. **Hacker News** - "Show HN: Lightning-fast goal tracker for OpenCode"
4. **Twitter/X** - Tag @opencode_ai
5. **Dev.to** - Write a blog post about it
6. **Product Hunt** - Launch as a product

## 📊 Tracking Usage

Add telemetry (optional, with user consent):

```go
// In main.go
const TelemetryEndpoint = "https://your-api.com/telemetry"

func reportUsage() {
    // Only if user opts in
    if !telemetryEnabled() {
        return
    }
    // Send anonymous usage stats
}
```

Or use GitHub stars/downloads as metrics.

## 🔒 Security

Before distributing:

1. **Code Review** - Ensure no sensitive data in code
2. **Dependencies** - Scan for vulnerabilities (`go mod verify`)
3. **Checksums** - Always provide SHA256 checksums
4. **Signatures** - Consider GPG signing releases
5. **HTTPS** - Use HTTPS for all download URLs

## 📝 Documentation

Ensure you have:

- [x] README.md - Overview and quick start
- [x] GETTING_STARTED.md - Detailed tutorial
- [x] LICENSE - MIT license
- [x] CHANGELOG.md - Version history
- [ ] CONTRIBUTING.md - How to contribute
- [ ] CODE_OF_CONDUCT.md - Community guidelines
- [ ] SECURITY.md - Security policy

## 🎯 Success Metrics

Track:
- GitHub stars
- Download count (from releases)
- Issues/PRs submitted
- Community feedback
- Forks

## 💡 Tips

1. **Start small** - Begin with GitHub releases
2. **Get feedback** - Share with friends first
3. **Document well** - Good docs = happy users
4. **Respond quickly** - Address issues promptly
5. **Iterate** - Release often with improvements
6. **Credit contributors** - Acknowledge help

## 🤝 Support

Provide support through:
- GitHub Issues
- GitHub Discussions
- Discord server (optional)
- Email (listed in README)

## Next Steps

1. **Create GitHub repository**
2. **Push code**
3. **Create first release (v1.0.0)**
4. **Test install script**
5. **Share with OpenCode community**

Good luck with your distribution! 🚀
