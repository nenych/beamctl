# Contributing to beamctl

Bug reports, fixes and new features are welcome. For anything larger than a small fix, please open an issue first so we can agree on the approach.

## Development

Requires Go 1.26 or newer. On macOS you also need the Xcode Command Line Tools (the transport uses cgo); on Windows nothing else.

```
go vet ./... && go test ./...
go build -o ~/.local/bin/beamctl .
```

Both platforms can be checked from either one: `GOOS=windows go vet ./...` compiles the Windows files on macOS.

The unit tests cover the protocol (report bytes, reply parsing, value mapping) and run without a light. Anything that touches the transport or adds a command has to be tried on a real Litra Beam LX light: say in the pull request what you tested and whether it was over Bluetooth, USB or both.

## Layout

| File | Contents |
|---|---|
| `main.go` | argument parsing and commands |
| `litra.go` | HID++ reports, reply parsing, value mapping |
| `presets.go` | `~/.config/beamctl/presets.json` |
| `hid_darwin.go`, `hid_windows.go` | transport: IOKit via cgo on macOS, the HID API through `syscall` on Windows |
| `picker_darwin.go`, `picker_windows.go` | the system colour picker behind `back pick` |

Platform-specific code goes in files with a `_<os>.go` suffix; each platform provides the same `request` and `pickColor` functions.

## Guidelines

- Keep pull requests small and focused on one change.
- Run `gofmt` and match the style of the surrounding code.
- The CLI uses only the Go standard library. Please discuss before adding a dependency.
- A new command needs a test that checks the exact bytes of its report.
- Frontends run the CLI and read its output, so do not change existing commands, output or exit codes without discussion.
- Project, binary and package names must not contain Logitech trademarks; refer to the light descriptively.

## Reporting a bug

Please include your operating system and its version, how the light is connected (Bluetooth or USB), the command you ran and its full output. On Windows, run it with `BEAMCTL_DEBUG=1` set.

## Conduct and licence

Be respectful and constructive. By contributing you agree that your contribution is licensed under the MIT License of this project.
