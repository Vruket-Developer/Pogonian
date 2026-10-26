# 🚀 Getting Started with Pogonian

This guide walks you through installing Pogonian, creating your first application, managing routes and database tables, configuring environment secrets, and building a production binary.

---

## 📋 Prerequisites

- **Go**: 1.22+ installed and available in your `PATH` (`go version`).
- **Git**: Installed for version control.
- **Node.js / npm** *(Optional)*: Only needed if you use external CSS bundlers. Pogonian's core runtime has **zero npm dependencies**.

---

## 📦 Installation

### Option 1: Run with `npx` (Zero Installation Required)

```bash
# Scaffold a new application directly
npx pogonian create store-app
cd store-app
npx pogonian dev
```

### Option 2: Install Globally via npm

```bash
npm install -g pogonian

# Use the 'pogn' or 'pogonian' command anywhere
pogn --version
```

### Option 3: Install via Go Toolchain

```bash
go install github.com/Vruket-Developer/Pogonian/cmd/pogn@latest
```

### Option 4: Build & Install from Source

Clone the repository and build the binary:

```bash
git clone https://github.com/Vruket-Developer/Pogonian.git
cd Pogonian
make build

# Install into your $GOPATH/bin
go install ./cmd/pogn
```

Verify the installation:

```bash
pogn --version
```

---

## 🏃 Quickstart: 0 to 1 in 60 Seconds

### Step 1: Scaffold a New Project

Run `pogn create` followed by your application name:

```bash
pogn create store-app
cd store-app
```

This creates a production-ready application layout:
```text
store-app/
├── .env                  # Cryptographic APP_KEY and local configs (git-ignored)
├── .env.example          # Environment template for team members
├── .gitignore            # Standard git ignore rules
├── pogonian.yaml         # Project configuration
├── pogonian.lock         # Framework lockfile
├── globals.css           # Global design system & variables
├── globals.js            # Global client-side scripts
├── app/                  # Frontend filesystem routes
│   ├── layout.html       # Root layout wrapper with header and navigation
│   ├── 404.html          # Custom 404 page
│   ├── page.html         # Homepage (/)
│   └── about/
│       └── page.html     # About page (/about)
└── server/               # 5-Tier Go backend modules
    └── users/            # Authentication & session services
```

---

### Step 2: Start the Development Server

Start the live dev server:

```bash
pogn dev
```

Output:
```text
  Pogonian Dev Server v0.1.0
  ➜  Local:   http://127.0.0.1:3000
  ➜  App:     store-app
  ➜  Routes:  3 discovered
```

Open `http://localhost:3000` in your browser.  
Notice:
- Clicking navigation links performs **instant soft-navigation** with zero full-page reload.
- The **DevBar HUD** is visible at the bottom right.
- You can click **⚡ DevTools** to see live architecture graphs, memory metrics, and AST representations.

---

### Step 3: Add Database Tables & Full-Stack Routes

To add a new entity (for example, `cart` or `products`):

```bash
pogn add table cart
```

This automatically scaffolds:
1. `server/cart/entity.go`: Go struct model
2. `server/cart/repository.go`: Parameterized SQL repository
3. `server/cart/service.go`: Business logic tier
4. `server/cart/handler.go`: HTTP form action handler
5. `app/cart/page.html`: Responsive HTML frontend view and dynamic input form

To customize fields, simply edit `server/cart/entity.go`:
```go
package cart

type CartItem struct {
    ID        int64     `json:"id" db:"id"`
    ItemName  string    `json:"item_name" db:"item_name"`
    Quantity  int       `json:"quantity" db:"quantity"`
    Price     float64   `json:"price" db:"price"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}
```

Then regenerate the frontend view matching the struct:
```bash
pogn add route /cart
```

---

### Step 4: Validate Architecture & System Health

Check layer boundaries and diagnostic health:

```bash
pogn check
pogn doctor
```

Output:
```text
🩺 Pogonian Environment & Project Doctor
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✓ Go Toolchain:          go version go1.22+ linux/amd64
  ✓ Pogonian CLI:          v0.1.0 (Architecture-First)
  ✓ Project Config:        store-app (v0.1.0, modular-monolith)
  ✓ Database (SQLite):     WAL mode active at ./data.db
  ✓ Server Port 3000:      Available for binding
  ✓ Filesystem Router:     4 routes discovered cleanly
  ✓ Security Posture:      Score 100/100 (Grade A+, 0 findings)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✨ System health check complete!
```

---

### Step 5: Configure Environment Variables

Edit your `.env` file to customize ports, database connections, or hostnames:

```env
APP_NAME=store-app
APP_ENV=development
APP_KEY=pogn_sec_9d4f820c7e2b10a56e9c4d3f1a8b0c7e2b10a56e9c4d3f1a8b0c7e2b10a56e9c
SESSION_SECRET=pogn_sec_e2b10a56e9c4d3f1a8b0c7e2b10a56e9c4d3f1a8b0c7e2b10a56e9c4d3f1a8b0

# Server settings
PORT=3000
HOST=127.0.0.1

# Database settings (SQLite local or remote PostgreSQL)
DATABASE_URL=sqlite://data.db
# DATABASE_URL=postgres://user:password@10.0.0.5:5432/storedb?sslmode=disable
```

To rotate your `APP_KEY` secret key at any time:
```bash
pogn secret
```

---

### Step 6: Compile for Production

Compile your application into a standalone, single-file binary:

```bash
pogn build
```

Output:
```text
📦 Building Pogonian Production Release: /path/to/store-app

  🔨 Compiling standalone optimized production binary: /path/to/store-app/dist/store-app

✨ Standalone production build complete!
   • Binary:    dist/store-app (14.2 MB)
   • Mode:      Zero external dependencies (Single Binary Release)
   • Run:       ./dist/store-app
```

Run your standalone production server:

```bash
./dist/store-app
```

Deploy this binary to any Linux server, Docker container, Fly.io, Railway, AWS, or GCP with zero setup.
