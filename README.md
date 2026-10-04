# beamctl

Command-line control for the Logitech Litra Beam LX light: front light, back RGB light and presets.

Works on macOS and Windows, over Bluetooth (`046D:B903`) and USB (`046D:C903`); USB is preferred when the light is reachable both ways.

## Install

**macOS** needs Go and the Xcode Command Line Tools. **Windows** needs only Go.

```
go install github.com/nenych/beamctl@latest
```

puts the binary in `~/go/bin` (`%USERPROFILE%\go\bin` on Windows). Or, from a clone on macOS:

```
go build -o ~/.local/bin/beamctl .
```

Frontends (such as `beamctl-mx-ring`) run in environments without your shell's `PATH`, so they look for the binary in `~/.local/bin`, `/opt/homebrew/bin`, `/usr/local/bin` and `~/go/bin`, in that order, and only then on `PATH`. Install it in one of those.

## Usage

```
beamctl status
beamctl on | off | toggle
beamctl brightness <0-100 | +N | -N>       percent of the 30-400 lm range
beamctl temp <2700-6500 | +N | -N>         kelvin, rounded to 100
beamctl back on | off | toggle
beamctl back brightness <1-100 | +N | -N>  percent
beamctl back color <RRGGBB>
beamctl back pick                          choose in the system colour picker, turns the back light on
beamctl preset [name]                      without a name, lists presets one per line
```

`+N` / `-N` change the current value. Exit status is 0 on success, 1 on a failure (message on stderr), 2 on a usage error.

## Presets

`~/.config/beamctl/presets.json` (`%USERPROFILE%\.config\beamctl\presets.json` on Windows); every field is optional, and applying a preset turns the affected lights on:

```json
{
  "call":  {"brightness": 60, "temp": 4500},
  "night": {"brightness": 20, "temp": 2700, "back": {"color": "ff6a00", "brightness": 40}}
}
```

## Troubleshooting on Windows

Set `BEAMCTL_DEBUG=1` to see which Logitech HID devices were found and which one was chosen:

```
set BEAMCTL_DEBUG=1
beamctl status
```

In PowerShell the first line is `$env:BEAMCTL_DEBUG=1`.

## Development

```
go vet ./... && go test ./...
```

The protocol is Logitech HID++ 2.0 in 20-byte reports (`litra.go`). The transport is IOKit via cgo on macOS (`hid_darwin.go`) and the Windows HID API on Windows (`hid_windows.go`).

Contributions are welcome, see [CONTRIBUTING.md](CONTRIBUTING.md).

## Acknowledgements

The HID++ commands for the Litra Beam LX were worked out by [litra-rs](https://github.com/timrogers/litra-rs) (USB) and [litra-ble-mqtt](https://github.com/Alevale/litra-ble-mqtt) (Bluetooth), both MIT-licensed. No code from them is included here.

## Licence and trademarks

MIT, see [LICENSE](LICENSE).

This project is not affiliated with, sponsored or endorsed by Logitech. Logitech, Logi, and their logos are trademarks or registered trademarks of Logitech Europe S.A. and/or its affiliates in the United States and/or other countries.
