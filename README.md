# beamctl

A command-line (CLI) tool for controlling a Logitech Litra Beam LX light from the terminal on macOS, Windows and Linux, over Bluetooth or USB. It switches the front light on and off, sets its brightness and colour temperature, controls the RGB backlight and applies saved presets.

```
beamctl on
beamctl brightness 60
beamctl temp 4500
beamctl back color ff6a00
```

beamctl is an independent open-source project and is not made by Logitech.

## Platforms

| | USB | Bluetooth |
|---|---|---|
| macOS | tested | tested |
| Windows | tested | tested |
| Linux | tested | not tried yet |

The light is `046D:C903` over USB and `046D:B903` over Bluetooth. When it is reachable both ways, USB is used.

## Install

**macOS** needs Go and the Xcode Command Line Tools. **Windows** and **Linux** need only Go.

```
go install github.com/nenych/beamctl@latest
```

puts the binary in `~/go/bin` (`%USERPROFILE%\go\bin` on Windows). Or, from a clone on macOS or Linux:

```
go build -o ~/.local/bin/beamctl .
```

Frontends such as [beamctl-mx-ring](https://github.com/nenych/beamctl-mx-ring) run in environments without your shell's `PATH`, so they look for the binary in `~/.local/bin`, `/opt/homebrew/bin`, `/usr/local/bin` and `~/go/bin`, in that order, and only then on `PATH`. Install it in one of those.

### Linux: access to the light

By default only root may open the light's `/dev/hidraw*` node. Install the udev rule from this repository once, then reconnect the light (unplug the cable, or disconnect and reconnect Bluetooth):

```
sudo cp 70-beamctl.rules /etc/udev/rules.d/
sudo udevadm control --reload && sudo udevadm trigger
```

The rule gives access to whoever is logged in at the machine itself, on its desktop or console. It does nothing for a remote login over SSH, a service or a cron job: for those, uncomment the two group lines at the end of the rule file before installing it, as described there.

`beamctl back pick` uses `zenity` or `kdialog` for its colour dialog; without either, use `beamctl back color RRGGBB`.

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

`back` is the RGB backlight. `+N` / `-N` change the current value. Exit status is 0 on success, 1 on a failure (message on stderr), 2 on a usage error.

## Presets

`~/.config/beamctl/presets.json` (`%USERPROFILE%\.config\beamctl\presets.json` on Windows); every field is optional, and applying a preset turns the affected lights on:

```json
{
  "call":  {"brightness": 60, "temp": 4500},
  "night": {"brightness": 20, "temp": 2700, "back": {"color": "ff6a00", "brightness": 40}}
}
```

## Troubleshooting on Windows and Linux

Set `BEAMCTL_DEBUG=1` to see which Logitech HID devices were found and which one was chosen. On Linux:

```
BEAMCTL_DEBUG=1 beamctl status
```

On Windows, in PowerShell (in `cmd` the first line is `set BEAMCTL_DEBUG=1`):

```
$env:BEAMCTL_DEBUG=1
beamctl status
```

## Related projects

- [beamctl-mx-ring](https://github.com/nenych/beamctl-mx-ring) puts these controls into the Actions Ring of a Logitech MX Master 4: a Logi Options+ plugin for macOS and Windows that runs beamctl for every action.

## Development

```
go vet ./... && go test ./...
```

The protocol is Logitech HID++ 2.0 in 20-byte reports (`litra.go`). The transport is IOKit via cgo on macOS (`hid_darwin.go`), the Windows HID API on Windows (`hid_windows.go`) and hidraw on Linux (`hid_linux.go`).

Contributions are welcome, see [CONTRIBUTING.md](CONTRIBUTING.md).

## Acknowledgements

The HID++ commands for the Litra Beam LX were worked out by [litra-rs](https://github.com/timrogers/litra-rs) (USB) and [litra-ble-mqtt](https://github.com/Alevale/litra-ble-mqtt) (Bluetooth), both MIT-licensed. No code from them is included here.

## Licence and trademarks

MIT, see [LICENSE](LICENSE).

This project is not affiliated with, sponsored or endorsed by Logitech. Logitech, Logi, and their logos are trademarks or registered trademarks of Logitech Europe S.A. and/or its affiliates in the United States and/or other countries.
