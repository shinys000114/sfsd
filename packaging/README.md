# Debian package

Build a local Debian package:

```sh
./build-deb.sh
```

The package version is based on the build script version format:

```text
TAG+YYMMDDHHMMSS~DIST
```

For Debian's `Version` field, a leading `v` tag prefix is removed. For example,
`v0.1.3+260629183014` for `focal` is packaged as:

```text
0.1.3+260629183014~focal
```

If there is no git tag, the package version is:

```text
0.0.0+dev.YYMMDDHHMMSS~DIST
```

Useful overrides:

```sh
BUILD_DATE=260629183014 ./build-deb.sh
DEB_ARCH=arm64 ./build-deb.sh
DEB_DIST=focal ./build-deb.sh
```

Build the same source for multiple Ubuntu distributions:

```sh
export DEB_MAINTAINER="sys114 <shinys000114@gmail.com>"
build_date=$(date +%y%m%d%H%M%S)
for dist in focal jammy noble resolute; do
    BUILD_DATE="$build_date" DEB_DIST="$dist" ./build-deb.sh
done
```

The build script generates `/usr/share/doc/sfsd/changelog.Debian.gz` inside the
package. Its distribution field is set from `DEB_DIST`, or from
`lsb_release -cs` when `DEB_DIST` is not provided.
