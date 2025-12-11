// Copyright 2025 goTap Authors. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package goTap

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	green   = "\033[97;42m"
	white   = "\033[90;47m"
	yellow  = "\033[90;43m"
	red     = "\033[97;41m"
	blue    = "\033[97;44m"
	magenta = "\033[97;45m"
	cyan    = "\033[97;46m"
	reset   = "\033[0m"
)

var (
	// DefaultWriter is the default io.Writer used by goTap for debug output and
	// middleware output like Logger() or Recovery().
	DefaultWriter io.Writer = os.Stdout

	// DefaultErrorWriter is the default io.Writer used by goTap to debug errors
	DefaultErrorWriter io.Writer = os.Stderr

	// goTapMode indicates current mode (debug, release, test).
	goTapMode = debugCode
	modeName  = "debug"
)

const (
	// DebugMode indicates goTap mode is debug.
	DebugMode = "debug"
	// ReleaseMode indicates goTap mode is release.
	ReleaseMode = "release"
	// TestMode indicates goTap mode is test.
	TestMode = "test"
)

const (
	debugCode = iota
	releaseCode
	testCode
)

// SetMode sets goTap mode according to input string.
func SetMode(value string) {
	if value == "" {
		value = DebugMode
	}

	switch value {
	case DebugMode:
		goTapMode = debugCode
	case ReleaseMode:
		goTapMode = releaseCode
	case TestMode:
		goTapMode = testCode
	default:
		panic("goTap mode unknown: " + value + " (available mode: debug release test)")
	}

	modeName = value
}

// Mode returns current goTap mode.
func Mode() string {
	return modeName
}

func debugPrint(format string, values ...any) {
	if IsDebugging() {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		fmt.Fprintf(DefaultWriter, "[goTap-debug] "+format, values...)
	}
}

func debugPrintWARNINGDefault() {
	if IsDebugging() {
		debugPrint(`[WARNING] Now goTap requires Go 1.21+.

`)
		debugPrint(`[WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

`)
	}
}

func debugPrintError(err error) {
	if err != nil && IsDebugging() {
		fmt.Fprintf(DefaultErrorWriter, "[goTap-debug] [ERROR] %v\n", err)
	}
}

// IsDebugging returns true if the framework is running in debug mode.
// Use SetMode(goTap.ReleaseMode) to disable debug mode.
func IsDebugging() bool {
	return goTapMode == debugCode
}
