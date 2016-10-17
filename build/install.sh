#!/bin/bash
# POSIX

# Static
PKG="github.com/pydio/pydio-booster"

pushd ${GOPATH}/src
find ${GOPATH}/src -name glide.yaml -not -path "*/vendor/*" -execdir glide install ${PKG} \;
go install ${PKG}/cmd/pydio
popd
