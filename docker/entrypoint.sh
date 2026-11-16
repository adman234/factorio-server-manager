#!/bin/bash
set -euo pipefail

FSM_DIR=/opt/fsm
FSM_DATA=/opt/fsm-data
CONF="$FSM_DATA/conf.json"
FACTORIO_DIR=/opt/factorio
FACTORIO_BIN="$FACTORIO_DIR/bin/x64/factorio"

log() { echo "[entrypoint] $*"; }

is_true() {
    case "${1,,}" in
        true | 1 | yes | on) return 0 ;;
        *) return 1 ;;
    esac
}

# Creates the config on the first start and applies the environment on every start.
update_config() {
    local secure=false tmp
    if is_true "${FSM_SECURE_COOKIES:-false}"; then
        secure=true
    fi

    if [ ! -f "$CONF" ]; then
        log "Creating $CONF"
        jq --arg db "$FSM_DATA/sqlite.db" --arg logfile "$FSM_DATA/factorio-server-manager.log" \
            '.sq_lite_database_file = $db | .log_file = $logfile' \
            "$FSM_DIR/conf.json" >"$CONF"
    fi

    tmp=$(mktemp "$CONF.XXXXXX")
    jq --argjson secure "$secure" --arg rcon "${RCON_PASS:-}" \
        '.secure = $secure | if $rcon != "" then .rcon_pass = $rcon else . end' \
        "$CONF" >"$tmp"
    chmod 600 "$tmp"
    mv "$tmp" "$CONF"

    if [ -n "${RCON_PASS:-}" ]; then
        log "RCON password set from RCON_PASS"
    fi
    log "Secure cookies: $secure"
}

installed_version() {
    if [ -x "$FACTORIO_BIN" ]; then
        "$FACTORIO_BIN" --version 2>/dev/null | sed -n 's/^Version: \([0-9][0-9.]*\).*/\1/p' | head -n 1 || true
    fi
}

# Prints the version FACTORIO_VERSION refers to, or nothing if it can't be resolved.
resolve_version() {
    local channel
    case "$FACTORIO_VERSION" in
        stable) channel=stable ;;
        latest | experimental) channel=experimental ;;
        *)
            echo "$FACTORIO_VERSION"
            return
            ;;
    esac
    curl -fsSL --retry 3 --connect-timeout 15 --max-time 60 https://factorio.com/api/latest-releases |
        jq -er --arg channel "$channel" '.[$channel].headless' || true
}

major_minor() { echo "$1" | cut -d. -f1-2; }

verify_checksum() {
    local version=$1 file=$2 sums expected
    if ! sums=$(curl -fsSL --retry 3 --connect-timeout 15 --max-time 60 https://factorio.com/download/sha256sums/); then
        log "WARNING: could not fetch Factorio checksums, skipping checksum verification"
        return 0
    fi
    expected=$(awk -v f="factorio-headless_linux_${version}.tar.xz" '$2 == f { print $1 }' <<<"$sums")
    if [ -z "$expected" ]; then
        log "WARNING: no published checksum for Factorio $version, skipping checksum verification"
        return 0
    fi
    echo "$expected  $file" | sha256sum -c --quiet -
}

install_factorio() {
    local version=$1 work
    work=$(mktemp -d)

    log "Downloading Factorio $version"
    if ! curl -fsSL --retry 3 --retry-delay 5 --connect-timeout 30 -o "$work/factorio.tar.xz" \
        "https://www.factorio.com/get-download/${version}/headless/linux64" ||
        ! verify_checksum "$version" "$work/factorio.tar.xz" ||
        ! tar -xJf "$work/factorio.tar.xz" -C "$work" ||
        [ ! -x "$work/factorio/bin/x64/factorio" ]; then
        rm -rf "$work"
        log "ERROR: downloading or unpacking Factorio $version failed"
        return 1
    fi

    # saves/, mods/ and config/ are volumes, so replace only the game files.
    rm -rf "${FACTORIO_DIR:?}/bin" "${FACTORIO_DIR:?}/data" "${FACTORIO_DIR:?}/config-path.cfg"
    cp -a "$work/factorio/." "$FACTORIO_DIR/"
    rm -rf "$work"
    log "Installed Factorio $(installed_version)"
}

ensure_factorio() {
    local installed target
    installed=$(installed_version)
    target=$(resolve_version)

    if [ -z "$target" ]; then
        if [ -n "$installed" ]; then
            log "WARNING: could not resolve FACTORIO_VERSION=$FACTORIO_VERSION, keeping installed Factorio $installed"
            return 0
        fi
        log "ERROR: could not resolve FACTORIO_VERSION=$FACTORIO_VERSION and Factorio is not installed"
        return 1
    fi

    if [ "$installed" = "$target" ]; then
        log "Factorio $installed is up to date"
        return 0
    fi

    # Tracking a channel must not silently cross a major release: saves and mods may not survive it.
    if [ -n "$installed" ] && [ "$target" != "$FACTORIO_VERSION" ] &&
        [ "$(major_minor "$installed")" != "$(major_minor "$target")" ] &&
        ! is_true "${FACTORIO_ALLOW_MAJOR_UPGRADE:-false}"; then
        log "WARNING: Factorio $target is available, but it is a major update from $installed."
        log "WARNING: Back up your saves, then set FACTORIO_ALLOW_MAJOR_UPGRADE=true or FACTORIO_VERSION=$target to update."
        return 0
    fi

    if ! install_factorio "$target"; then
        if [ -n "$installed" ]; then
            log "WARNING: keeping installed Factorio $installed"
            return 0
        fi
        return 1
    fi
}

update_config
ensure_factorio

cmd=("$FSM_DIR/factorio-server-manager"
    --conf "$CONF"
    --dir "$FACTORIO_DIR"
    --port 80
    --autostart="${FSM_AUTOSTART:-false}"
    "$@")

cd "$FSM_DIR"

if [ -n "${PUID:-}" ]; then
    PGID=${PGID:-$PUID}
    log "Running as $PUID:$PGID"
    chown -R "$PUID:$PGID" "$FSM_DATA" "$FACTORIO_DIR" "$FSM_DIR/mod_packs"
    exec setpriv --reuid="$PUID" --regid="$PGID" --clear-groups -- "${cmd[@]}"
fi

exec "${cmd[@]}"
