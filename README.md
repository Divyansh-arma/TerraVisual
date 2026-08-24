# Terra Visual 🌍⚡

[![Release](https://img.shields.io/github/v/release/divyansh/terra-visual?color=0284c7&label=Release)](https://github.com/divyansh/terra-visual/releases)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Windows%20%7C%20Linux-indigo.svg)](https://github.com/divyansh/terra-visual/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Terra Visual** is a local-first, privacy-focused desktop application that turns your Terraform code and state into an interactive, editable visual architecture canvas. Detect infrastructure drift, audit security vulnerabilities, and edit your architecture visually with bi-directional code synchronization.

---

## 🔒 100% Local-First & Air-Gapped

> **Your infrastructure code and state files NEVER leave your computer.**
> All AST parsing, state inspection, drift reconciliation, security evaluation, and visual rendering execute strictly on `localhost`. No cloud accounts required, zero telemetry, and no remote servers.

---

## ✨ Key Features

### 🎨 Interactive Architecture Canvas
- Render complex multi-tier cloud architectures into clean, interactive directed graphs.
- Hierarchical auto-layout with Top-to-Bottom and Left-to-Right orientation toggles.
- Nested container subgraphs for **VPCs**, **Subnets**, and **Virtual Networks**.
- Real-time search filter and interactive pan, zoom, and minimap controls.

### 🔄 Bi-Directional Visual Editing (Graph $\leftrightarrow$ HCL Code)
- Add new cloud resources or delete existing nodes directly on the canvas.
- Click **"Sync to Code"** to automatically update your `.tf` files using Abstract Syntax Tree (AST) manipulation.
- Preserves comments, existing variable references, and code formatting.

### 🔍 State vs. Code Drift Detection
- Ingest your `terraform.tfstate` and `.tf` files simultaneously to uncover drift:
  - 🟢 **`IN_SYNC`**: Provisioned state perfectly matches declared HCL code.
  - 🟡 **`MODIFIED`**: Attributes or dependency links have drifted.
  - 🔵 **`MISSING_IN_STATE`**: Resource declared in code but not yet deployed.
  - 🔴 **`MISSING_IN_CODE`**: Resource exists in live state but was deleted from code.

### 🛡️ Automated Local Security Scanning
- Real-time detection of security misconfigurations and compliance violations.
- Visual warning shields with severity badges (CRITICAL, HIGH, MEDIUM, LOW) and hover tooltips.
- Inspect remediation advice and rule IDs directly in the node inspector drawer.

### ☁️ Multi-Cloud Resource Catalog
- Built-in template registry for **AWS**, **Azure**, and **Google Cloud (GCP)**.
- Pre-configured services across **Networking**, **Compute**, **Databases**, **Storage**, and **Load Balancers**.
- Customize resource labels and configuration attributes with an integrated JSON editor.

---

## 📥 How to Install

1. Go to the [**Releases**](https://github.com/divyansh/terra-visual/releases) tab.
2. Download the installer for your operating system:
   - **macOS**: Download `Terra-Visual.dmg` (Universal for Apple Silicon & Intel).
   - **Windows**: Download `Terra-Visual-Setup.exe` or `.msi`.
   - **Linux**: Download `Terra-Visual.AppImage` or `.deb`.
3. Run the installer and launch **Terra Visual**.

---

## 🎯 How to Use Terra Visual

1. **Select State File**: Click **"Select State File"** to pick your `terraform.tfstate`.
2. **Select Code Directory**: Click **"Select Code Directory"** to choose your Terraform `.tf` project folder.
3. **Analyze**: Click **"Run Drift Analysis"** (or **"Parse HCL Code"**) to generate the interactive canvas.
4. **Edit Visually**:
   - Click **"Add Resource"** to drop new cloud services onto the canvas.
   - Select any node and press <kbd>Delete</kbd> or <kbd>Backspace</kbd> to remove it.
5. **Sync**: Click **"Sync to Code"** to write your changes back to your local `.tf` files.

---

## 🤝 Contributing & Developer Guide

Interested in building Terra Visual from source or contributing features? See [**CONTRIBUTING.md**](CONTRIBUTING.md) for full developer setup, architecture details, and test suites.

---

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.
