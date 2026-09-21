# tuneup

[![CI](https://github.com/MuhammadZa1/tuneup/actions/workflows/ci.yml/badge.svg)](https://github.com/MuhammadZa1/tuneup/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**A diagnostic tool for Linux machines that Linux itself seems to have forgotten about.**

`tuneup` checks for the specific, real problems that show up on aging or
low-spec hardware — the kind of stuff that eats a whole evening of
forum-diving before you find the fix. It's read-only by default and
only ever changes your system when you explicitly say so.

I built the first version of this on a 2012 MacBook Pro running Linux
Mint, after hitting every one of these problems myself.

Available as a **CLI** (this README) or as a **desktop app** with a
sidebar, per-check tabs, light/dark themes, a system tray icon,
launch-on-login, and scheduled background scans with notifications —
see [`app/`](app/) for the app, or grab a pre-built `.deb`/`.AppImage`
from [Releases](../../releases/latest).

## What it checks

| Check | What it catches |
|---|---|
| **Audio crackling** | PipeWire's default scheduling quantum being too small for slower CPUs, causing crackling/dropouts |
| **GPU / Vulkan support** | Vulkan silently falling back to a CPU software renderer (lavapipe) instead of real GPU hardware — the reason games "launch" and then crawl or crash |
| **Kernel / dkms mismatch** | dkms modules (often Wi-Fi or GPU drivers) that failed to build for the current or a newly installed kernel — before a reboot leaves you without networking |
| **Swap / zram** | Low-RAM machines with no swap or zram configured, which turns memory pressure into freezes or the OOM killer instead of graceful slowdown |
| **Battery wear** | How much capacity a laptop battery has lost against its original design capacity, plus cycle count where available |
| **Disk I/O scheduler** | Spinning HDDs left on a scheduler poorly suited to seek-heavy access patterns, instead of bfq — fixable, and persists via a udev rule |
| **Thermal throttling** | Confirmed CPU throttle events since boot (via the kernel's own counters) or sustained high temperatures likely to cause throttling |
| **Wi-Fi power saving** | Power management enabled on a wireless interface, a common cause of intermittent drops/laggy pings on some chipsets — fixable, and persists via NetworkManager when available |

More checks are planned — see [Roadmap](#roadmap).

## Install

**Download a pre-built binary (no Go required):**

Grab the archive for your OS/architecture from the [latest release](../../releases/latest), extract it, and run the `tuneup` binary directly:

```bash
tar -xzf tuneup-linux-amd64.tar.gz
./tuneup
```

**Or install with Go (requires Go 1.21+):**

```bash
go install github.com/MuhammadZa1/tuneup/cmd/tuneup@latest
```

**Or build from source:**

```bash
git clone https://github.com/MuhammadZa1/tuneup.git
cd tuneup
go build -o tuneup ./cmd/tuneup
```

## Usage

```bash
tuneup
```

```
tuneup — diagnostics for aging Linux hardware
Read-only checks first. Nothing is changed without your confirmation.

! Audio crackling (PipeWire)
  No custom scheduling quantum is set. On slower CPUs this is a common
  cause of audio crackling/dropouts.
  Fix available: Write a quantum override to /etc/pipewire/pipewire.conf.d/99-tuneup-quantum.conf (will ask for your password) and restart PipeWire
  Apply this fix? [y/N]
```

Every check runs read-only first. If a check finds something it knows
how to safely fix, it asks before touching anything. Fixes that need
elevated permissions prompt for your password via `pkexec`, which
works whether you're running from a terminal or from the desktop app.

**Flags:**

| Flag | Effect |
|---|---|
| `--yes` | Apply every offered fix without prompting (useful for scripts/CI on your own fleet) |
| `--only <names>` | Run only the given comma-separated check names instead of all of them, e.g. `--only disk-scheduler,battery-wear` |
| `--json` | Print results as a single JSON object instead of interactive text — no prompts. Combine with `--yes` to also apply fixes and report the outcome per-check |
| `--no-color` | Disable colored output |
| `--version` | Print the version and exit |

## Design principles

- **Read-only by default.** Every check only inspects the system.
  Nothing is written or changed unless you confirm a specific fix (or
  pass `--yes`).
- **No fix for things with no safe default.** Some findings (GPU
  hardware limits, dkms build failures, battery wear) are surfaced as
  information, not "fixes," because there either isn't a safe
  automated action or the right answer depends on your hardware and
  preferences, not a one-size-fits-all script.
- **Honest about what it can't check.** If a required tool isn't
  installed, the check reports `SKIPPED` and says why — it never
  guesses.

## Roadmap

- [ ] Distro-aware fixes (currently Debian/Ubuntu-family focused)

## Contributing

Issues and PRs are welcome — especially new checks for problems you've
hit on your own hardware. A check is just a small Go file implementing
one interface; see `internal/checks/` for examples.

## License

[MIT](LICENSE)
