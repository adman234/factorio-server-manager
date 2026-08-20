[![.github/workflows/test-workflow.yml](https://github.com/OpenFactorioServerManager/factorio-server-manager/workflows/.github/workflows/test-workflow.yml/badge.svg)](https://github.com/OpenFactorioServerManager/factorio-server-manager/actions)
[![Discord](https://img.shields.io/discord/779512040934342687?label=Discord)](https://discord.gg/SB647WmSbU)

# Factorio Server Manager (adman234's fork)

> **This is an AI slop repo.** These changes were written and applied by Claude (Anthropic's AI) at the request of the repo owner, who was debugging their own self-hosted Factorio server and asked for the fixes below. Code was traced through and reasoned about, and where possible validated (the UI build was run, the Go changes were checked against an existing accepted upstream PR for the same bug), but it has **not been reviewed by a human developer**, and only limited testing was possible in the environment this was written in (no Go toolchain, no Docker, no live container run). Review before you trust it with anything that matters.

Forked from [OpenFactorioServerManager/factorio-server-manager](https://github.com/OpenFactorioServerManager/factorio-server-manager), which has been without a release for about 5 years. This fork cherry-picks a few fixes that were sitting unmerged upstream, plus a couple of small additions:

- **Fixed `FSM_AUTOSTART` not working.** The Docker entrypoint never actually passed `--autostart` to the binary, relying entirely on an environment-variable fallback that didn't reliably take effect. Now threaded through explicitly. ([upstream issue #301](https://github.com/OpenFactorioServerManager/factorio-server-manager/issues/301), [issue #409](https://github.com/OpenFactorioServerManager/factorio-server-manager/issues/409))
- **Fixed `autostart` being impossible to set via `conf.json`.** The config field was tagged `json:"-"`, which silently excluded it from ever being read from or written to the config file, despite the project's own wiki documenting that as a supported method. (Same fix as unmerged upstream [PR #332](https://github.com/OpenFactorioServerManager/factorio-server-manager/pull/332).)
- **Fixed the Server Settings form not submitting** — a stale `react-hook-form` register callback on the visibility checkboxes. (Cherry-picked from unmerged upstream [PR #417](https://github.com/OpenFactorioServerManager/factorio-server-manager/pull/417).)
- **Security: bumped `golang.org/x/crypto`** 0.14.0 → 0.17.0 and **`follow-redirects`** 1.15.1 → 1.15.6, both addressing known CVEs. (Cherry-picked from Dependabot PRs [#379](https://github.com/OpenFactorioServerManager/factorio-server-manager/pull/379) and [#384](https://github.com/OpenFactorioServerManager/factorio-server-manager/pull/384) that were never merged.)
- **Added a favicon** (an original gear icon, not Wube's trademarked Factorio logo) — the web UI didn't have one at all.
- **Added a GitHub Actions workflow** to build and publish the Docker image to `ghcr.io` on push, so this fork can be built and run without relying on the original's Docker Hub account.

### Running this fork's image

```bash
docker run -d --name FactorioServer \
  -e FSM_AUTOSTART=true \
  -p 9999:80/tcp \
  -p 34197:34197/udp \
  -v /path/to/appdata/fsm_saves:/opt/factorio/saves \
  -v /path/to/appdata/fsm_mods:/opt/factorio/mods \
  -v /path/to/appdata/fsm_config:/opt/factorio/config \
  -v /path/to/appdata/fsm_data:/opt/fsm-data \
  ghcr.io/adman234/fsm:latest
```

Volume paths match the original image's layout (`/opt/factorio/...` plus `/opt/fsm-data`), so pointing these at an existing FSM data directory should pick up saves/mods/config as-is.

### A tool for managing Factorio servers.
This tool runs on a Factorio server and allows management of the Factorio server, saves, mods and many other features.

## Features
* Allows control of the Factorio Server, starting and stopping the Factorio binary.
* Allows the management of save files, upload, download and delete saves.
* Manage installed mods, upload new ones and more
* Manage modpacks, so it is easier to play with different configurations
* Allow viewing of the server logs and current configuration.
* Authentication for protecting against unauthorized users
* Available as a Docker container

#### Manage Factorio Server
![Factorio Server Manager Screenshot](screenshots/Screenshot_Controls.png)

#### Manage save files
![Factorio Server Manager Screenshot](screenshots/Screenshot_Saves.png)

#### Manage mods
![Factorio Server Manager Screenshot](screenshots/Screenshot_Mods.png)

## [Installation and Usage](https://github.com/OpenFactorioServerManager/factorio-server-manager/wiki/Installation-and-Usage)

## [Development](https://github.com/OpenFactorioServerManager/factorio-server-manager/wiki/Development)

## Contributing
1. Fork it!
2. Checkout the develop branch, only use that as a base: `git checkout develop`
2. Create your feature branch: `git checkout -b my-new-feature`
3. Commit your changes: `git commit -am 'Add some feature'`
4. Add your changes a in human readable way into CHANGELOG.md
4. Push to the branch: `git push origin my-new-feature`
5. Submit a pull request, with `develop` as base :D

## Authors

* **Mitch Roote** - [roote.ca](https://roote.ca)
* **[knoxfighter](https://github.com/knoxfighter)**
* **[Jannaahs](https://github.com/jannaahs)**

## Special Thanks
- **[All Contributions](https://github.com/OpenFactorioServerManager/factorio-server-manager/graphs/contributors)**
- **mickael9** for reverseengineering the factorio-save-file: https://forums.factorio.com/viewtopic.php?f=5&t=8568#

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
