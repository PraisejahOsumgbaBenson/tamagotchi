#!/usr/bin/env bash
#
# Fetch the Tamagotchi P1 ROM and verify it before use.
#
# The ROM is Bandai's copyright and is not distributed with this project.
# Point ROM_URL at a copy you have obtained, or run `make rom` with the URL
# set, and this script downloads it, checks the hash, and writes it to
# roms/rom.bin.
#
# A wrong ROM produces a screen full of nonsense and no error message, so the
# hash check here is the difference between a five minute fix and a lost
# afternoon.

set -euo pipefail

ROM_SHA1="4b4979cf92dc9d2fb6d7295a38f209f3da144f72"
ROM_MD5="3fce172403c27274e59723bbb9f6d8e9"
ROM_SIZE=32768

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
rom_dir="${repo_root}/roms"
rom_path="${rom_dir}/rom.bin"

log() { printf '%s\n' "$*" >&2; }
die() { log "error: $*"; exit 1; }

verify() {
	local file="$1"

	local sha1 md5 size
	sha1="$(shasum -a 1 "$file" | awk '{print $1}')"
	md5="$(md5 -q "$file")"
	size="$(wc -c <"$file" | tr -d ' ')"

	if [[ "$sha1" != "$ROM_SHA1" ]]; then
		die "SHA1 mismatch for $file
  expected $ROM_SHA1
  actual   $sha1
This is not the Tamagotchi P1 ROM."
	fi

	if [[ "$md5" != "$ROM_MD5" ]]; then
		die "MD5 mismatch for $file (expected $ROM_MD5, actual $md5)"
	fi

	if [[ "$size" != "$ROM_SIZE" ]]; then
		log "warning: ROM is $size bytes, expected $ROM_SIZE"
	fi
}

extract() {
	local file="$1"
	local dir="$2"

	case "$file" in
	*.zip)
		unzip -o -q "$file" -d "$dir"
		;;
	*)
		cp "$file" "$dir/rom.bin"
		;;
	esac
}

main() {
	if [[ -z "${ROM_URL:-}" ]]; then
		die "ROM_URL is not set.
Get the Tamagotchi P1 ROM yourself and point ROM_URL at it, for example:
  ROM_URL=\"https://example.invalid/tama.zip\" make rom
See docs/ROM.md for where to look and what the file should hash to."
	fi

	if [[ ! -f "$rom_path" ]]; then
		mkdir -p "$rom_dir"
		local tmp
		tmp="$(mktemp -d)"
		trap 'rm -rf "$tmp"' EXIT

		log "downloading $ROM_URL"
		curl --fail --location --silent --show-error --output "$tmp/rom.download" "$ROM_URL" \
			|| die "download failed"

		extract "$tmp/rom.download" "$tmp"
		rm -f "$tmp/rom.download"

		local candidate
		candidate="$(find "$tmp" -type f ! -name '*.zip' -size +1k | head -n 1)"
		[[ -n "$candidate" ]] || die "no ROM found inside the download"

		log "verifying $(basename "$candidate")"
		verify "$candidate"

		mv "$candidate" "$rom_path"
		log "wrote $rom_path"
	else
		log "verifying existing $rom_path"
		verify "$rom_path"
		log "$rom_path looks good"
	fi
}

main "$@"
