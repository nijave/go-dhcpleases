#!/usr/bin/env bash

set -ex

cwd=$(pwd)
rm -rf build || true
mkdir build
go build
cp go-dhcpleases build/

pushd /usr/plugins/devel
cp -a "$cwd/plugin" go-dhcpleases
pushd go-dhcpleases
mkdir -p src/bin
cp "$cwd/go-dhcpleases" src/bin
chmod +x src/bin/*

make package

cp work/pkg/*.pkg "$cwd/build"
