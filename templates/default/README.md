# {{ .AppName }}

An AI-native, high-performance web application built with **Pogonian** (Go + HTML-first SPA Runtime).

---

## ⚡ Quick Start

```bash
# Start live development server (with Swagger UI & Live Reload)
pogn dev

# Open Swagger API documentation
pogn swagger --open

# Add new backend tables or routes
pogn add table cart
pogn add route /cart

# Validate architecture boundaries & security posture
pogn check
pogn doctor

# Build optimized standalone production release binary
pogn build
```

---

## 📁 Clean Project Structure

```
.
├── app/                  # Single source of truth for frontend & pages
│   ├── components/       # UI components (e.g. Card.html, footer.html)
│   ├── about/            # /about filesystem route
│   ├── products/         # /products dynamic entity route
│   ├── layout.html       # Root HTML layout with soft navigation
│   └── page.html         # Home page
├── docs/                 # Project documentation & architecture guides
├── public/               # Static assets (images, icons, fonts)
├── globals.css           # Global design system & theme tokens
├── globals.js            # Client-side extensions & custom scripts
├── pogonian.yaml         # Framework configuration
└── go.mod                # Go dependencies
```

---

## 📚 Documentation

Detailed guides are located in the [`docs/`](docs/) directory:
- [**CLI Commands Guide**](docs/CLI_COMMANDS.md)
- [**Architecture & Design System**](docs/ARCHITECTURE.md)
- [**Database & Migrations Guide**](docs/DATABASE_AND_MIGRATIONS.md)
- [**Getting Started & Releases**](docs/GETTING_STARTED.md)
