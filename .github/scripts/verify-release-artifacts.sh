#!/bin/sh

set -eu

dist_dir="${1:-dist}"
checksums="$dist_dir/checksums.txt"
cask="$dist_dir/homebrew/Casks/ollygarden.rb"

fail() {
    printf 'Error: %s\n' "$*" >&2
    exit 1
}

[ -f "$checksums" ] || fail "missing checksum manifest: $checksums"
[ -f "$cask" ] || fail "missing generated Homebrew cask: $cask"

archive_count=0
release_version=""
seen_archives=""

while read -r checksum archive; do
    [ -n "$checksum" ] || continue
    [ -f "$dist_dir/$archive" ] || fail "checksum manifest references missing archive: $archive"
    printf '%s\n' "$seen_archives" | grep -Fxq "$archive" && fail "duplicate release archive: $archive"
    seen_archives="${seen_archives}${archive}
"

    if command -v sha256sum >/dev/null 2>&1; then
        actual_checksum="$(sha256sum "$dist_dir/$archive" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
        actual_checksum="$(shasum -a 256 "$dist_dir/$archive" | awk '{print $1}')"
    else
        fail "missing required tool: sha256sum or shasum"
    fi
    [ "$checksum" = "$actual_checksum" ] || fail "checksum mismatch for $archive"

    case "$archive" in
        ollygarden_*_darwin_amd64.tar.gz|ollygarden_*_darwin_arm64.tar.gz|ollygarden_*_linux_amd64.tar.gz|ollygarden_*_linux_arm64.tar.gz|ollygarden_*_windows_amd64.zip|ollygarden_*_windows_arm64.zip)
            ;;
        *) fail "unexpected release archive: $archive" ;;
    esac

    version="${archive#ollygarden_}"
    version="${version%_darwin_amd64.tar.gz}"
    version="${version%_darwin_arm64.tar.gz}"
    version="${version%_linux_amd64.tar.gz}"
    version="${version%_linux_arm64.tar.gz}"
    version="${version%_windows_amd64.zip}"
    version="${version%_windows_arm64.zip}"

    if [ -z "$release_version" ]; then
        release_version="$version"
    elif [ "$release_version" != "$version" ]; then
        fail "release archives contain multiple versions: $release_version and $version"
    fi

    case "$archive" in
        *_darwin_*.tar.gz|*_linux_*.tar.gz)
            archive_prefix="ollygarden_${release_version}_"
            cask_archive="${archive#"$archive_prefix"}"
            awk -v checksum="$checksum" -v archive="$cask_archive" '
                /^[[:space:]]+on_(intel|arm) do$/ { has_checksum = 0; has_url = 0 }
                index($0, "sha256 \"" checksum "\"") { has_checksum = 1 }
                index($0, "ollygarden_#{version}_" archive) { has_url = 1 }
                has_checksum && has_url { found = 1 }
                END { exit !found }
            ' "$cask" || fail "cask URL and checksum do not match $archive"
            ;;
    esac

    archive_count=$((archive_count + 1))
done < "$checksums"

[ "$archive_count" -eq 6 ] || fail "expected 6 release archives, found $archive_count"

for suffix in \
    darwin_amd64.tar.gz \
    darwin_arm64.tar.gz \
    linux_amd64.tar.gz \
    linux_arm64.tar.gz \
    windows_amd64.zip \
    windows_arm64.zip
do
    expected_archive="ollygarden_${release_version}_$suffix"
    printf '%s\n' "$seen_archives" | grep -Fxq "$expected_archive" || fail "missing release archive: $expected_archive"
done

grep -Fq "version \"$release_version\"" "$cask" || fail "cask version does not match release archives"
grep -Fq 'homepage "https://github.com/ollygarden/ollygarden-cli"' "$cask" || fail "cask homepage is not canonical"
grep -Fq 'verified: "github.com/ollygarden/ollygarden-cli/"' "$cask" || fail "cask download origin is not verified"
