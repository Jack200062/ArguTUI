# ArguTUI - Terminal UI for ArgoCD

![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
![Status: Active](https://img.shields.io/badge/Status-Active-green.svg)

**ArguTUI** is a modern terminal user interface for ArgoCD that provides a feature-rich TUI client for interacting with your ArgoCD instances directly from the terminal.

## Features

- **Multi-Instance Management** - Connect to and switch between multiple ArgoCD instances
- **Application Overview** - List all ArgoCD applications with sync and health status information
- **Resource Management** - View and navigate through Kubernetes resources for each application
- **Tree-Based Resource View** - View resource dependencies in a tree structure with expand/collapse functionality
- **Powerful Filtering** - Filter by project, health status, sync status, and resource types
- **Quick Navigation** - Intuitive keyboard shortcuts for core actions
- **Search** - Fast search through applications and resources
- **Refresh and Sync** - Update status and force sync applications directly from the interface

## Installation

### Binary Releases

The easiest way to install ArguTUI is to download the pre-built binaries from the [Releases](https://github.com/Jack200062/ArguTUI/releases) page.

```bash
# Download the latest release (replace X.Y.Z with the latest version)
curl -L https://github.com/Jack200062/ArguTUI/releases/download/vX.Y.Z/argutui_X.Y.Z_linux_amd64.tar.gz | tar xz

# Move to a directory in your PATH
sudo mv argutui /usr/local/bin/
```

### From Source

```bash
# Prerequisites: Go 1.16 or higher

# Clone the repository
git clone https://github.com/Jack200062/ArguTUI.git
cd ArguTUI

# Build the application
go build -o bin/argutui ./cmd/argocd-tui

# Run
./bin/argutui
```

## Getting Started

1. **Prepare Configuration File**

   Create a `config.yml` file with your ArgoCD instance details:

   ```yaml
   instances:
     - name: production
       url: https://argocd.production.example.com
       token: <your-argocd-api-token>
       insecureskipverify: false
   ```

   By default, ArguTUI looks for this file at `config/config.yml`, but you can specify a different location with the `CONFIG_PATH` environment variable:

   ```bash
   export CONFIG_PATH=/path/to/your/config.yml
   ```

2. **Obtain ArgoCD API Token**

   You'll need an API token from ArgoCD:

   ```bash
   # Using ArgoCD CLI
   argocd account generate-token

   # Or via the API
   curl -X POST -d '{"username":"admin","password":"password"}' https://argocd.example.com/api/v1/session
   ```

3. **Launch ArguTUI**

   ```bash
   argutui
   ```

   If you have multiple instances configured, you'll be presented with a selection screen. Otherwise, ArguTUI will connect directly to the single instance defined in your configuration.

## Configuration Details

### Config File Structure

```yaml
instances:
  - name: prod                          # A friendly name for the instance
    url: https://argocd.example.com     # ArgoCD API server URL
    token: eyJhbGciOiJIUzI1NiIsInR5...  # Your ArgoCD API token
    insecureskipverify: false           # Whether to skip TLS verification

  - name: dev
    url: https://argocd-dev.example.com
    token: abcdefghijklmnopqrstuvwxyz
    insecureskipverify: true
```

### Configuration Options

- `name`: A descriptive name for your ArgoCD instance
- `url`: The URL of your ArgoCD API server
- `token`: Your ArgoCD API token
- `insecureskipverify`: Set to `true` to skip TLS certificate verification (useful for development environments)

## Key Shortcuts

### Global

| Key           | Action                     |
|---------------|----------------------------|
| <kbd>q</kbd>  | Quit application           |
| <kbd>?</kbd>  | Show help                  |
| <kbd>b</kbd>  | Go back                    |
| <kbd>/</kbd>  | Search in current view     |
| <kbd>I</kbd>  | Return to instance select  |

### Applications Screen

| Key              | Action                    |
|------------------|---------------------------|
| <kbd>Enter</kbd> | Open application resources|
| <kbd>R</kbd>     | Refresh all applications  |
| <kbd>r</kbd>     | Refresh selected app      |
| <kbd>S</kbd>     | Sync application          |
| <kbd>D</kbd>     | Delete application        |
| <kbd>f, F</kbd>  | Show filter menu          |
| <kbd>c, C</kbd>  | Clear all filters         |

### Resources Screen

| Key           | Action                     |
|---------------|----------------------------|
| <kbd>Enter</kbd> | Expand/collapse resource |
| <kbd>t</kbd>  | Toggle all expansions      |
| <kbd>f, F</kbd> | Show filter menu         |

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

*Made with ❤️ by [Jack200062](https://github.com/Jack200062)*
