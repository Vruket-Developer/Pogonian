# 🧩 Pogonian Marketplace, Modules & UI Components

Pogonian features a **Shadcn-style component and module registry**.

Instead of pulling opaque `npm` dependencies that bundle heavy JavaScript runtimes, `pogn add module` / `pogn add component` copies clean, readable, native HTML5 templates, CSS, Go backend code, and database migrations directly into your project repository. **You own and control 100% of the code.**

---

## 🚀 Quick Start

### 1. View Available Marketplace Components
```bash
pogn modules
# or
pogn marketplace
```

### 2. Add a UI Component (e.g. Footer, Header, Sidebar, Card)
```bash
pogn add component footer
# or
pogn add module footer
```

**What this does:**
1. Installs the component template into `app/components/footer.html`.
2. Appends clean, scoped CSS into `globals.css`.
3. Auto-registers the component with Pogonian's HTML compiler so you can use `{{ template "footer" . }}` in any layout or page!

---

## 🎨 How to Include Components in Your Pages

Pogonian's template compiler automatically discovers all templates in `app/components/*.html`.

### Method A: In Global Layout (`app/layout.html`)
To display the footer across all pages, include it in your root layout:
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{ .Title }} | {{ .AppName }}</title>
    <link rel="stylesheet" href="/globals.css">
</head>
<body class="pogn-body">
    <!-- Header Component -->
    {{ template "header" . }}

    <main id="pogn-root" class="container main-content">
        {{ .Content }}
    </main>

    <!-- Footer Component -->
    {{ template "footer" . }}
</body>
</html>
```

### Method B: On Specific Pages (`app/about/page.html`)
```html
<section class="content-section">
    <h1>About Us</h1>
    <p>Welcome to our company portal.</p>
    
    <!-- Include Card Component -->
    {{ template "card" . }}
</section>
```

---

## ⚡ Full-Stack Feature Modules (UI + Go Backend + DB + Tests)

When you install a full-stack module like `login`, Pogonian scaffolds both the frontend views and the entire Go backend:

```bash
pogn add module login
```

**Scaffolded Artifacts:**
- `app/login/page.html` (Accessible Sign In page with CSRF protection)
- `app/register/page.html` (User registration page)
- `server/auth/` (Go Handler, Service, Repository, and Entity with Argon2id hashing)
- `migrations/001_create_login.up.sql` (`users` & `sessions` SQL schema)
- `migrations/001_create_login.down.sql` (Rollback script)

---

## 📦 What About React, Angular, NPM & Client Libraries?

### Can I still use Shadcn / React / Angular with Pogonian?
**Yes!** Pogonian is HTML-first and runs a native Go server, but you have full freedom over your frontend stack:
1. **Native Pogonian Components (Recommended)**: Zero build steps, instant server rendering, lightweight (<10KB JS runtime), perfect SEO, and 0 bundle bloat.
2. **NPM & Client-side Libraries**: You can run `npm install <package>` in your Pogonian project to use Tailwind CSS, Lucide icons, Shoelace Web Components, Alpine.js, HTMX, or React Islands whenever you need client-side widgets.

---

## 🛠️ Removing Components or Modules
```bash
# Remove a UI component
pogn remove component footer

# Remove a full-stack module
pogn remove module login
```
