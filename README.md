# Factorio Server Manager (adman234's fork)

A web UI for running a Factorio multiplayer server: start and stop it, and manage
saves, mods, modpacks, logs and settings.

This is a fork of [OpenFactorioServerManager/factorio-server-manager](https://github.com/OpenFactorioServerManager/factorio-server-manager),
created by Mitch Roote and maintained by knoxfighter and Jannaahs (see [Authors](#authors)).
Upstream has had no commits since July 2024 and no release for several years, so
this fork collects fixes that were left unmerged there and publishes its own image.
If upstream becomes active again these changes can be offered back.

The changes were written with Claude and have had limited testing. Review them
before relying on them.

## What is different from upstream

- **`FSM_AUTOSTART` works in Docker.** The entrypoint now passes `--autostart` to the
  binary instead of relying on an environment fallback that did not take effect.
  ([#301](https://github.com/OpenFactorioServerManager/factorio-server-manager/issues/301), [#409](https://github.com/OpenFactorioServerManager/factorio-server-manager/issues/409))
- **`autostart` can be set in `conf.json`.** The field was tagged `json:"-"`, so it was
  never read from the file. Same fix as unmerged [PR #332](https://github.com/OpenFactorioServerManager/factorio-server-manager/pull/332).
- **The Server Settings form submits again.** Cherry-picked from unmerged
  [PR #417](https://github.com/OpenFactorioServerManager/factorio-server-manager/pull/417).
- **Security updates** for `golang.org/x/crypto` (0.14.0 to 0.17.0) and `follow-redirects`
  (1.15.1 to 1.15.6), from Dependabot PRs [#379](https://github.com/OpenFactorioServerManager/factorio-server-manager/pull/379)
  and [#384](https://github.com/OpenFactorioServerManager/factorio-server-manager/pull/384).
- **A favicon** (an original gear icon, not the Factorio logo).
- **Images on GHCR** built by GitHub Actions, instead of upstream's Docker Hub account.

## Running it

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

## Screenshots

#### Manage Factorio Server
![Factorio Server Manager Screenshot](screenshots/Screenshot_Controls.png)

#### Manage save files
![Factorio Server Manager Screenshot](screenshots/Screenshot_Saves.png)

#### Manage mods
![Factorio Server Manager Screenshot](screenshots/Screenshot_Mods.png)

## Documentation

Upstream's wiki still applies: [Installation and Usage](https://github.com/OpenFactorioServerManager/factorio-server-manager/wiki/Installation-and-Usage), [Development](https://github.com/OpenFactorioServerManager/factorio-server-manager/wiki/Development).

## Authors

Original project authors:

* **Mitch Roote** - [roote.ca](https://roote.ca)
* **[knoxfighter](https://github.com/knoxfighter)**
* **[Jannaahs](https://github.com/jannaahs)**

## Special Thanks
- **[All Contributions](https://github.com/OpenFactorioServerManager/factorio-server-manager/graphs/contributors)**
- **mickael9** for reverse engineering the factorio-save-file: https://forums.factorio.com/viewtopic.php?f=5&t=8568#

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
