# Push this repo to your GitHub account

Use this once to create the **public** repo and push from your machine.

## 0. Git identity (if needed)

If you haven’t set your name/email for Git, do it once (use your GitHub email and name):

```bash
git config --global user.email "you@example.com"
git config --global user.name "Your Name"
```

Then create the initial commit (if not done yet):

```bash
cd /path/to/pi-terminal-world-monitor-client
git add .gitignore AGENTS.md README.md backend/ client/ contrib/ docs/ openspec/
git commit -m "Initial commit: backend (FastAPI + health), client (blessed TUI), Pi/DietPi install and startup docs"
```

## 1. Create the repo on GitHub

**Option A – GitHub website**

1. Go to [github.com/new](https://github.com/new).
2. **Repository name:** `pi-terminal-world-monitor-client` (or e.g. `pi-world-monitor`).
3. **Public**, no need to add README / .gitignore (we already have them).
4. Click **Create repository**.

**Option B – GitHub CLI**

If you have [gh](https://cli.github.com/) installed and logged in:

```bash
gh repo create pi-terminal-world-monitor-client --public --source=. --remote=origin --push
```

If you use `--push` it will push the current branch. Otherwise continue with step 2.

## 2. Add remote and push (if you didn’t use `gh repo create --push`)

Replace `YOUR_USERNAME` with your GitHub username:

```bash
cd /home/eymardfreire/pi-terminal-world-monitor-client
git remote add origin https://github.com/YOUR_USERNAME/pi-terminal-world-monitor-client.git
git push -u origin main
```

If you chose a different repo name, use that in the URL.

## 3. After first push

- **Install on Pi:** Use the clone URL from your new repo in [docs/INSTALL-PI.md](INSTALL-PI.md) (step 1).  
- **Clone on another machine:** `git clone https://github.com/YOUR_USERNAME/pi-terminal-world-monitor-client.git`
