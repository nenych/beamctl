## Install

- **Homebrew** (macOS, Linux): `brew install nenych/tap/beamctl`
- **Script** (macOS, Linux): `curl -fsSL https://raw.githubusercontent.com/nenych/beamctl/main/install.sh | sh`
- **Windows:** unzip `beamctl_windows_amd64.zip` (or `arm64`) and put `beamctl.exe` in a folder on your `PATH`, for example `%USERPROFILE%\.local\bin`.
- **Any platform with Go:** `go install github.com/nenych/beamctl@latest`

The macOS binaries are not signed by Apple, so one downloaded here with a browser is blocked by Gatekeeper; Homebrew and the script do not have that problem. On Linux, install `70-beamctl.rules` from the archive as described in the [README](https://github.com/nenych/beamctl#linux-access-to-the-light).

`checksums.txt` holds the SHA-256 of every archive.
