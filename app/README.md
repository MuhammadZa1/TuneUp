# tuneup-app

The desktop app for `tuneup` — a graphical front end with a sidebar,
per-check tabs, light/dark themes, a system tray icon, launch-on-login,
and scheduled background scans with notifications.

It bundles the `tuneup` CLI as a sidecar binary and calls it with
`--json` under the hood. This app never re-implements check logic —
it's purely a UI on top of the same binary the CLI ships.

## Install

Grab the `.deb` or `.AppImage` for your system from the
[latest release](../../releases/latest) — no separate `tuneup` install
needed, the CLI binary is bundled inside.

**Debian/Ubuntu/Mint (.deb):**
```
sudo apt install ./tuneup-app_*.deb
```

**AppImage (any distro):**
```
chmod +x tuneup-app_*.AppImage
./tuneup-app_*.AppImage
```

## Features

- Dashboard view showing all checks at once, or scan any single check
  from its own tab
- Light and dark themes
- Optional system tray icon — closing the window minimizes to tray
  instead of quitting
- Optional launch on login
- Optional scheduled background scans with a desktop notification when
  something's found

## Building from source

**Requirements:**

- **Rust via rustup** (not your distro's `rustc`/`cargo` package),
  version 1.77+. Distro-packaged Rust is often too old for current
  crate versions — install it properly with:
  ```
  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
  ```
  Then restart your terminal (or `source $HOME/.cargo/env`).

- **System libraries** (Debian/Ubuntu/Mint — this targets the modern
  webkit2gtk-4.1/libsoup3 stack that current Ubuntu/Mint ships):
  ```
  sudo apt update
  sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev libayatana-appindicator3-dev librsvg2-dev build-essential curl pkg-config
  ```

- **Tauri CLI (v2):**
  ```
  cargo install tauri-cli --version "^2" --locked
  ```

**Run it in development**, from the `tuneup-app` folder:
```
cargo tauri dev
```
This opens the app window directly, using the sidecar binary already
in `src-tauri/binaries/`.

**Build the real installable app:**
```
cargo tauri build
```
This produces a `.deb` and an `.AppImage` under
`src-tauri/target/release/bundle/`.

**Updating the bundled tuneup binary:** if you change the `tuneup` CLI
itself, rebuild it and copy it back in before running `cargo tauri
build` again:
```
cd /path/to/tuneup
go build -ldflags "-s -w -X main.version=X.Y.Z" -o /path/to/tuneup-app/src-tauri/binaries/tuneup-x86_64-unknown-linux-gnu ./cmd/tuneup
```
The filename must keep the `-x86_64-unknown-linux-gnu` suffix — that's
how Tauri matches sidecar binaries to the build target.
