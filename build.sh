#!/bin/sh

now=$(date +'%Y-%m-%d_%T')
go build -a -ldflags "-linkmode external -extldflags '-static' -s -w -X main.sha1ver=$SOURCE_VERSION -X main.buildTime=$now"  -o main main.go
