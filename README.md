# Cupertino

The MacOS package manager.

Browse packages at [cupertino.sh](https://cupertino.sh).

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/arnavsurve/cupertino/main/install.sh | bash
```

## Usage

```bash
# Install a package
cupertino install <package>

# Install a specific version
cupertino install <package>@<version>

# Install from a local tarball
cupertino install ./mypackage.tar.gz

# List installed packages
cupertino list

# Search for packages
cupertino search <query>

# Show package details
cupertino info <package>

# Uninstall a package
cupertino uninstall <package>

# Upgrade packages
cupertino upgrade [package]

# Skip confirmation prompts
cupertino install -y <package>
```

## Package format

Packages are `.tar.gz` archives containing a `package.json` manifest:

```json
{
  "name": "mytool",
  "version": "1.0.0",
  "description": "A useful tool",
  "license": "MIT",
  "dependencies": {
    "libfoo": ">=2.0.0"
  },
  "files": {
    "bin/mytool": "bin/mytool"
  }
}
```

Dependency constraints support `>=`, `^`, `~`, exact versions, and `*` (any).
