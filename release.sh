#!/bin/bash

MODULE=$(grep module go.mod | cut -d\  -f2)
BINBASE=${MODULE##*/}
VERSION=${VERSION:-$GITHUB_REF_NAME}
COMMIT_HASH="$(git rev-parse --short HEAD)"
BUILD_TIMESTAMP=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
BUILDER=$(go version)

[ -d dist ] && rm -rf dist
mkdir dist

# For version in sub module
# "-X '${MODULE}/main.Version=${VERSION}'"

LDFLAGS=(
  "-X 'main.Version=${VERSION}'"
  "-X 'main.CommitHash=${COMMIT_HASH}'"
  "-X 'main.BuildTimestamp=${BUILD_TIMESTAMP}'"
  "-X 'main.Builder=${BUILDER}'"
)

echo "[*] go get"
go get .

echo "[*] go builds:"
for DIST in {linux,openbsd,windows,freebsd}/{amd64,arm,arm64}; do
#for DIST in linux/{amd64,386}; do
  GOOS=${DIST%/*}
  GOARCH=${DIST#*/}
  echo "[+]   $DIST:"
  echo "[-]    - build"
  SUFFIX=""
  [ "$GOOS" = "windows" ] && SUFFIX=".exe"
  TARGET=${BINBASE}-${GOOS}-${GOARCH}
  env GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="${LDFLAGS[*]}" -o dist/${TARGET}${SUFFIX}
  echo "[-]    - compress"
  if [ "$GOOS" = "windows" ]; then
    (cd dist; zip -qm9 ${TARGET}.zip ${TARGET}${SUFFIX})
  else
    xz dist/${TARGET}
  fi
done

echo "[*] sha256sum"
(cd dist; sha256sum *) | tee ${BINBASE}.sha256sum
mv ${BINBASE}.sha256sum dist/

#echo "[*] pack"
#tar -cvf all.tar -C dist/ . && mv all.tar dist

echo "[*] done"
