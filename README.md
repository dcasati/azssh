# azssh

Connect to [Azure Cloud Shell](https://docs.microsoft.com/en-us/azure/cloud-shell/overview) from your terminal

[![Build Status](https://dev.azure.com/noelbundick/noelbundick/_apis/build/status/azssh?branchName=master)](https://dev.azure.com/noelbundick/noelbundick/_build/latest?definitionId=27?branchName=master)

## Installation

### Using Go

```bash
go install github.com/noelbundick/azssh@latest
```

### From Source

```bash
git clone https://github.com/noelbundick/azssh.git
cd azssh
go build
```

## Usage

* Launch with `azssh`
* Exit by typing `exit`
* Choose your shell: `azssh --shell bash` or `azssh --shell pwsh`

> Note: This app uses the clientId from the [vscode-azure-account](https://github.com/microsoft/vscode-azure-account) Visual Studio Code extension in order to call the necessary APIs. You will be prompted to allow access to "Visual Studio Code". This is expected behavior.

## Requirements

* Go 1.21 or later
* Azure subscription with Cloud Shell enabled

## Development

```bash
# Clone the repository
git clone https://github.com/noelbundick/azssh.git
cd azssh

# Install dependencies
go mod download

# Build
go build

# Run tests
go test ./...
```