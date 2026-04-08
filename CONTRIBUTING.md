# 🤝 Contributing to Linuxus

Thank you for your interest in contributing to **Linuxus**.

This project is designed to provide a web-based Ubuntu shell environment for education, and we welcome contributions of all kinds — code, documentation, and ideas.

---

## 🚀 Getting Started

1. Fork the repository
2. Clone your fork:

   ```bash
   git clone https://github.com/<your-username>/linuxus
   cd linuxus
   ```

3. Create a new branch:

   ```bash
   git checkout -b feature/your-feature-name
   ```

---

## 🧩 Contribution Types

We use the following PR types.
All pull requests must start with one of these prefixes:

| Type        | Description                                    |
| ----------- | ---------------------------------------------- |
| `[CHORE]`   | Maintenance / config / build changes           |
| `[REFACTOR]`| Code structure improvement (no behavior change)|
| `[DOCS]`    | Documentation update                           |
| `[FEATURE]` | Minor feature addition                         |
| `[GENESIS]` | Major update / architectural change            |
| `[BUG]`     | Bug fix                                        |
| `[DUP]`     | Duplicate or redundant PR                      |

### ✅ Example PR titles

```
[DOCS] update README usage section
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

## 🧪 Running the Project

```bash
./util/simple_build_and_run.sh -g -u
```

---

## 🧱 Code Style Guidelines

* Keep code simple and readable
* Prefer explicit over implicit logic
* Avoid unnecessary abstraction
* Follow existing project structure

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
* Duplicate or conflicting PRs may be marked as `[DUP]`
* Maintainers may request changes before merging

---

## 💬 Communication

* Be respectful and constructive
* Focus on technical discussion
* Keep feedback concise and clear

---

## 📄 License

By contributing to this project, you agree that your contributions will be licensed under the [MIT License](./LICENSE).
