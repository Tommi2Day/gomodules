//go:build tools
// +build tools

// Package tools contains all tools, which we need to
// to vendor and which are used to build the actual
// app binary
package tools

import (
	// blank imports to make sure `go mod vendor`
	// will download all dependencies
	_ "github.com/boumenot/gocover-cobertura"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/jstemmer/go-junit-report/v2"
)
