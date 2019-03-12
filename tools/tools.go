// +build tools

package tools

import (
	// Import all of the external tools to trick go modules into downloading them
	// and not tidying them.
	_ "github.com/AlekSi/gocov-xml"
	_ "github.com/axw/gocov/gocov"
	_ "github.com/client9/misspell/cmd/misspell"
	_ "github.com/golang/mock/mockgen"
	_ "github.com/matm/gocov-html"
	_ "golang.org/x/lint/golint"
	_ "golang.org/x/tools/cmd/goimports"
)
