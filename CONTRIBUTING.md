# 🤝 Contributing to Linuxus

Thank you for your interest in contributing to **Linuxus**.

This project is designed to provide a web-based Ubuntu shell environment for education, and we welcome contributions of all kinds — code, documentation, and ideas.

---

## 🚀 Getting Started

1. Fork the repository
2. Clone your fork:

   ```bash
   git clone https://github.com/<YOUR_GITHUB_NAME>/linuxus
   cd linuxus
   ```

3. Create a new branch:

   ```bash
   git checkout -b <YOUR_GITHUB_NAME>/<SIMPLE_PR_TITLE>
   ```

---

## 🧩 Contribution Types

We use the following PR types.
All pull requests must start with one of these prefixes:

| Type          | Description                                     |
|---------------|-------------------------------------------------|
| `[CHORE]`     | Maintenance / config / build changes            |
| `[REFACTOR]`  | Code structure improvement (no behavior change) |
| `[DOCUMENT]`  | Documentation update                            |
| `[FEATURE]`   | Minor feature addition                          |
| `[GENESIS]`   | Major update / architectural change             |
| `[BUG]`       | Bug fix                                         |
| `[DUPLICATE]` | Duplicate or redundant PR                       |

### ✅ Example PR titles

```
[DOCUMENT] update README usage section
[BUG] fix login session issue
[FEATURE] add logout endpoint
[GENESIS] redesign container lifecycle system
```

---

## 🔀 Pull Request Guidelines

Before submitting a PR:

* Make sure your PR title follows the required format (`[TYPE] ...`)
* Ensure the project builds and runs correctly
* Keep changes minimal and focused
* Update documentation if needed

### 📋 Checklist

* [ ] PR title uses correct prefix
* [ ] Code compiles and runs
* [ ] No unnecessary files included
* [ ] Related documentation updated (if applicable)
* [ ] Changes tested locally

---

## 🧱 Code Style Guidelines

* Keep code simple and readable
* Prefer explicit over implicit logic
* Avoid unnecessary abstraction
* Follow existing project structure

---

## Administrator UI checks

Run the Go regression suite with `go -C src test -race ./...`.
The browser checks use Node 20 or newer, Playwright, and its Chromium browser:

```bash
node scripts/test_admin_ui.mjs
```

The checks use simulated API responses and do not access deployed services or
user data. Set `PLAYWRIGHT_MODULE` to the absolute path of an existing Playwright
installation when it is not available through normal Node module resolution.
`PLAYWRIGHT_BROWSERS_PATH` selects a custom browser installation directory.
Set `LINUXUS_UI_SCREENSHOTS` to save desktop, mobile, and dialog screenshots.
These tools are only needed for development; the deployed binary embeds the
administrator HTML, CSS, and JavaScript.

---

## 🐛 Reporting Issues

If you find a bug:

* Use the **Bug Report template**
* Provide steps to reproduce
* Include logs if possible

---

## 💡 Suggesting Features

* Use the **Feature Request template**
* Clearly explain the motivation
* Avoid overly broad or vague ideas

---

## 🔄 Workflow Overview

Typical contribution flow:

```text
fork → branch → commit → PR → review → merge
```

---

## ⚠️ Important Notes

* Large changes (`[GENESIS]`) should be discussed in an issue first
* Duplicate or conflicting PRs may be marked as `[DUPLICATE]`
* Maintainers may request changes before merging

---

## 💬 Communication

* Be respectful and constructive
* Focus on technical discussion
* Keep feedback concise and clear

---

## 📄 License

By contributing to this project, you agree that your contributions will be licensed under the [MIT License](./LICENSE).
