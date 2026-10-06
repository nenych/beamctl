# Contributing to beamctl

Bug reports, fixes and new features are welcome. For anything larger than a small fix, please open an issue first so we can agree on the approach.

## Development

Requires Go 1.24 or newer; an older Go, from 1.21 on, downloads the right toolchain by itself. On macOS you also need the Xcode Command Line Tools (the transport uses cgo); on Windows and Linux nothing else.

```
go vet ./... && go test ./...
go build -o ~/.local/bin/beamctl .
```

Every platform can be checked from any other: `GOOS=windows go vet ./...` and `GOOS=linux go vet ./...` compile the other platforms' files. The Linux transport has tests of its own, which run only on Linux; from macOS or Windows use Docker:

```
docker run --rm -v "$PWD":/src:ro -w /src -e GOFLAGS=-buildvcs=false golang:1.26 go test ./...
```

The unit tests cover the protocol (report bytes, reply parsing, value mapping) and run without a light. Anything that touches the transport or adds a command has to be tried on a real Litra Beam LX light: say in the pull request what you tested and whether it was over Bluetooth, USB or both.

## Layout

| File | Contents |
|---|---|
| `main.go` | argument parsing and commands |
| `litra.go` | HID++ reports, reply parsing, value mapping |
| `presets.go` | `~/.config/beamctl/presets.json` |
| `hid_darwin.go`, `hid_windows.go`, `hid_linux.go` | transport: IOKit via cgo on macOS, the HID API through `syscall` on Windows, hidraw on Linux |
| `picker_darwin.go`, `picker_windows.go`, `picker_linux.go` | the colour picker behind `back pick` |
| `70-beamctl.rules` | udev rule that gives the logged-in user access to the light on Linux |

Platform-specific code goes in files with a `_<os>.go` suffix; each platform provides the same `request` and `pickColor` functions.

## Guidelines

- Keep pull requests small and focused on one change.
- Run `gofmt` and match the style of the surrounding code.
- The CLI uses only the Go standard library. Please discuss before adding a dependency.
- A new command needs a test that checks the exact bytes of its report.
- Frontends run the CLI and read its output, so do not change existing commands, output or exit codes without discussion.
- Project, binary and package names must not contain Logitech trademarks; refer to the light descriptively.

## Releasing

Push a tag like `v1.2.3`. The `Release` workflow tests, builds the archives for macOS, Linux and Windows with `scripts/build-release.sh`, publishes a GitHub release with `checksums.txt`, and points the formula in [nenych/homebrew-tap](https://github.com/nenych/homebrew-tap) at the tag (`scripts/update-tap.sh`, which needs the `HOMEBREW_TAP_TOKEN` secret; without it, update the formula by hand). Running the workflow by hand builds the archives without publishing anything.

## Reporting a bug

Please include your operating system and its version, how the light is connected (Bluetooth or USB), the command you ran and its full output. On Windows, run it with `BEAMCTL_DEBUG=1` set.

## Conduct and licence

Be respectful and constructive. By contributing you agree that your contribution is licensed under the MIT License of this project.
