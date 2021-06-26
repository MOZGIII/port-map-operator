#!/bin/bash
set -euo pipefail

for MOD in $(go list -mod=readonly -m -f '{{ if and (not .Indirect) (not .Main)}}{{.Path}}{{end}}' all); do
  ! go get "$MOD"
done

go mod tidy
go mod vendor
