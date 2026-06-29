#!/bin/bash
set -euo pipefail

ROOT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
. "$ROOT_DIR/scripts/version.sh"

PKG_NAME=${PKG_NAME:-sfsd}
DEB_ARCH=${DEB_ARCH:-$(dpkg --print-architecture 2>/dev/null || echo amd64)}
DEB_DIST=${DEB_DIST:-$(lsb_release -cs 2>/dev/null || echo unstable)}
MAINTAINER=${DEB_MAINTAINER:-"sfsd maintainers <root@localhost>"}
if [[ ! "$DEB_DIST" =~ ^[a-z0-9][a-z0-9.+-]*$ ]]; then
    echo "Invalid Debian distribution name: $DEB_DIST" >&2
    exit 1
fi
DEB_VERSION="${DEB_VERSION}~${DEB_DIST}"
OUT_DIR="$ROOT_DIR/dist"
BUILD_ROOT="$OUT_DIR/deb/${PKG_NAME}_${DEB_VERSION}_${DEB_ARCH}"
DEB_OUT="$OUT_DIR/${PKG_NAME}_${DEB_VERSION}_${DEB_ARCH}.deb"

case "$DEB_ARCH" in
    amd64)
        GOARCH=amd64
        GOARM=
        ;;
    arm64)
        GOARCH=arm64
        GOARM=
        ;;
    armhf)
        GOARCH=arm
        GOARM=7
        ;;
    armel)
        GOARCH=arm
        GOARM=5
        ;;
    *)
        echo "Unsupported Debian architecture: $DEB_ARCH" >&2
        exit 1
        ;;
esac

echo "Building Debian package version: $DEB_VERSION"
echo "Binary version: $VERSION"
echo "Distribution: $DEB_DIST"
echo "Architecture: $DEB_ARCH"

rm -rf "$BUILD_ROOT"
install -d "$BUILD_ROOT/DEBIAN"
install -d "$BUILD_ROOT/usr/bin"
install -d "$BUILD_ROOT/usr/share/doc/$PKG_NAME"
install -d "$BUILD_ROOT/lib/systemd/system"
install -d "$BUILD_ROOT/etc/sfsd"

export CGO_ENABLED=0
export GOOS=linux
export GOARCH
if [ -n "$GOARM" ]; then
    export GOARM
else
    unset GOARM || true
fi

go build -ldflags="-X main.version=$VERSION -s -w" -o "$BUILD_ROOT/usr/bin/sfsd" "$ROOT_DIR/main.go"

install -m 0644 "$ROOT_DIR/LICENSE" "$BUILD_ROOT/usr/share/doc/$PKG_NAME/copyright"
install -m 0644 "$ROOT_DIR/README.md" "$BUILD_ROOT/usr/share/doc/$PKG_NAME/README.md"
install -m 0644 "$ROOT_DIR/packaging/debian/config.yaml" "$BUILD_ROOT/etc/sfsd/config.yaml"
install -m 0644 "$ROOT_DIR/packaging/debian/sfsd.service" "$BUILD_ROOT/lib/systemd/system/sfsd.service"
install -m 0755 "$ROOT_DIR/packaging/debian/postinst" "$BUILD_ROOT/DEBIAN/postinst"
install -m 0755 "$ROOT_DIR/packaging/debian/postrm" "$BUILD_ROOT/DEBIAN/postrm"

cat > "$BUILD_ROOT/usr/share/doc/$PKG_NAME/changelog.Debian" <<EOF
$PKG_NAME ($DEB_VERSION) $DEB_DIST; urgency=medium

  * Build $PKG_NAME $VERSION for $DEB_DIST.

 -- $MAINTAINER  $(date -R)
EOF
gzip -n -9 "$BUILD_ROOT/usr/share/doc/$PKG_NAME/changelog.Debian"

cat > "$BUILD_ROOT/DEBIAN/control" <<EOF
Package: $PKG_NAME
Version: $DEB_VERSION
Section: web
Priority: optional
Architecture: $DEB_ARCH
Maintainer: $MAINTAINER
Depends: ca-certificates, adduser
Description: Lightweight static file serving server
 sfsd is a lightweight static file server with virtual host, HTTP/3,
 compression, cache rule, directory rendering, and repository serving support.
EOF

cat > "$BUILD_ROOT/DEBIAN/conffiles" <<EOF
/etc/sfsd/config.yaml
EOF

dpkg-deb --build --root-owner-group "$BUILD_ROOT" "$DEB_OUT"
echo "Build Complete: $DEB_OUT"
