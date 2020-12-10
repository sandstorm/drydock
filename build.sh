#!/bin/bash

set -ex

go build main.go

mkdir -p ~/.docker/cli-plugins
rm -f ~/.docker/cli-plugins/docker-execroot
ln -s `pwd`/main ~/.docker/cli-plugins/docker-execroot

rm -f ~/.docker/cli-plugins/docker-vscode
ln -s `pwd`/main ~/.docker/cli-plugins/docker-vscode