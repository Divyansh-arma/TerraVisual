# Contributing to Terra Visual 🛠️

Thank you for your interest in contributing to **Terra Visual**! This document provides all developer guides, build instructions, and test procedures for local development and contributions.

---

## 🏗️ Architecture Overview

```
┌────────────────────────────────────────────────────────┐
│               Terra Visual Desktop App                 │
├────────────────────────────────────────────────────────┤
│  Frontend (React 18 + TypeScript + Tailwind CSS)       │
│  - React Flow Canvas (@xyflow/react)                   │
│  - Dagre Compound Hierarchical Layout Engine           │
│  - Multi-Cloud Resource Catalog Modal                  │
│  - Node Inspector & Security Misconfigurations Drawer  │
├────────────────────────────────────────────────────────┤
│  Desktop Bridge (Tauri v2 + Rust)                      │
│  - IPC Command Handlers (parse_state, parse_code, etc.)│
│  - Native File & Directory Dialogs                     │
│  - Cross-Platform Sandboxed Process Execution          │
├────────────────────────────────────────────────────────┤
│  Core Engine (Go CLI - `terra-parser`)                 │
│  - State Parser: terraform-json + provider normalizer  │
│  - Code Parser: HashiCorp hcl/v2 AST parser            │
│  - Drift Engine: Deep attribute & dependency matcher   │
│  - Sync Engine: HashiCorp hclwrite AST generator       │
│  - Security Engine: Trivy / Checkov JSON scanner       │
└────────────────────────────────────────────────────────┘
```

---

## 💻 Prerequisites

Ensure you have the following installed on your machine:
- **Go** (v1.22+): `brew install go`
- **Node.js** (v18+ or v20+): `brew install node`
- **Rust** (stable): `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh`
- **Trivy** (optional, for security scanning tests): `brew install trivy`

---

## 🚀 Setting Up Local Development

### 1. Clone the Repository

```bash
git clone https://github.com/divyansh/terra-visual.git
cd Terra-visual
```

### 2. Build the Go Parser Binary

```bash
cd terra-parser
go test -v ./...
go build -o ./bin/terra-parser ./cmd/terra-parser

# Copy binary to Tauri desktop assets
mkdir -p ../terra-visual-ui/src-tauri/binaries
cp ./bin/terra-parser ../terra-visual-ui/src-tauri/binaries/
cd ..
```

### 3. Install Frontend Dependencies & Start Dev Server

```bash
cd terra-visual-ui
npm install

# Start Tauri desktop app with Vite hot-reloading
npm run tauri dev
```

---

## 🧪 Running Test Suites

### Go Parser & Security Test Suite

```bash
cd terra-parser
go test -v -count=1 ./...
```

### Frontend TypeScript & Production Build

```bash
cd terra-visual-ui
npm run build
```

### Rust Desktop Backend Check

```bash
cd terra-visual-ui/src-tauri
cargo check
```

---

## 📦 Building Desktop Packages

To build the native desktop bundle (.dmg on macOS, .msi on Windows, .AppImage on Linux):

```bash
cd terra-visual-ui
npm run tauri build
```

---

## 📜 Pull Request Guidelines

1. Ensure all Go unit tests pass: `go test -v ./...`.
2. Ensure TypeScript compiles without errors: `npm run build`.
3. Verify that `cargo check` compiles in `src-tauri`.
4. Follow the local-first rule: Never introduce telemetry or cloud API calls that leak infrastructure state or code.
