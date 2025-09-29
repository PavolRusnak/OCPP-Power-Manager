# Prerequisites for OCPP Power Manager

Before running Step 1 (repo scaffold), install and verify these tools on your machine:

1. **Git**
   - Download: https://git-scm.com/downloads
   - Verify: `git --version`

2. **Go (>=1.22)**
   - Download: https://go.dev/dl/
   - Windows: install `go1.22.x.windows-amd64.msi`
   - Verify: `go version`

3. **Goose (DB migrations)**
   - Install: `go install github.com/pressly/goose/v3/cmd/goose@latest`
   - Verify: `goose --version`

4. **Node.js + npm** (for web UI build later)
   - Download: https://nodejs.org (use LTS version)
   - Verify: `node -v` and `npm -v`

5. **SQLite** (optional but useful for inspecting local DB)
   - Download: https://www.sqlite.org/download.html
   - Verify: `sqlite3 --version`

6. **Docker** (optional, only if using Postgres locally)
   - Download: https://www.docker.com/get-started
   - Verify: `docker --version`

---

## Installation order

1. Git  
2. Go  
3. Goose  
4. Node.js + npm  
5. SQLite  
6. Docker (optional)

---

## Quick verification

After installing, open a new terminal and run:

```
git --version
go version
goose --version
node -v
npm -v
sqlite3 --version   (optional)
docker --version    (optional)
```

All commands should print a version number before continuing to Step 1.
