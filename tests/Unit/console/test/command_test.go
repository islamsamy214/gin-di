package unit

import (
	"slices"
	"strings"
	"testing"
	"web-app/app/console"
)

// Only the pure argument builder and the non-executing flag paths are covered
// here: calling Handle with real arguments would shell out to `go test` and
// recurse into this very suite.

func TestGoTestArgsDefaults(t *testing.T) {
	args := console.GoTestArgs(console.TestOptions{})

	if args[0] != "test" {
		t.Errorf("args[0] = %q, want \"test\"", args[0])
	}

	if args[len(args)-1] != "./..." {
		t.Errorf("last arg = %q, want \"./...\"", args[len(args)-1])
	}

	// Caching would make the command report a stale pass.
	if !slices.Contains(args, "-count=1") {
		t.Errorf("args = %v, want -count=1 by default", args)
	}

	// -json is what the reporter consumes, so it must always be present.
	if !slices.Contains(args, "-json") {
		t.Errorf("args = %v, want -json by default", args)
	}

	for _, unwanted := range []string{"-v", "-race", "-cover", "-run"} {
		if slices.Contains(args, unwanted) {
			t.Errorf("args = %v, want no %s by default", args, unwanted)
		}
	}
}

func TestGoTestArgsFlags(t *testing.T) {
	tests := []struct {
		name    string
		options console.TestOptions
		want    string
	}{
		{name: "raw uses -v", options: console.TestOptions{Raw: true}, want: "-v"},
		{name: "race", options: console.TestOptions{Race: true}, want: "-race"},
		{name: "coverage", options: console.TestOptions{Coverage: true}, want: "-cover"},
		{name: "coverage adds coverpkg", options: console.TestOptions{Coverage: true}, want: "-coverpkg=./..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := console.GoTestArgs(tt.options)

			if !slices.Contains(args, tt.want) {
				t.Errorf("args = %v, want it to contain %s", args, tt.want)
			}
		})
	}
}

// --cache opts back into the toolchain's result cache.
func TestGoTestArgsCacheDropsCountOne(t *testing.T) {
	args := console.GoTestArgs(console.TestOptions{Cache: true})

	if slices.Contains(args, "-count=1") {
		t.Errorf("args = %v, want no -count=1 when caching is allowed", args)
	}
}

func TestGoTestArgsFilterBecomesRun(t *testing.T) {
	const filter = "TestLoginWithFactoryUser"

	args := console.GoTestArgs(console.TestOptions{Filter: filter})

	index := slices.Index(args, "-run")
	if index == -1 {
		t.Fatalf("args = %v, want -run", args)
	}

	// -run takes the pattern as the next argument, not as -run=pattern.
	if index+1 >= len(args) || args[index+1] != filter {
		t.Errorf("args = %v, want %q directly after -run", args, filter)
	}
}

// Extra arguments must reach the toolchain, but never after the package pattern.
func TestGoTestArgsPassesExtraThrough(t *testing.T) {
	args := console.GoTestArgs(console.TestOptions{Extra: []string{"-timeout=30s", "-shuffle=on"}})

	for _, extra := range []string{"-timeout=30s", "-shuffle=on"} {
		index := slices.Index(args, extra)
		if index == -1 {
			t.Fatalf("args = %v, want it to contain %s", args, extra)
		}

		if index > slices.Index(args, "./...") {
			t.Errorf("args = %v, want %s before the package pattern", args, extra)
		}
	}
}

func TestGoTestArgsCombinesEverything(t *testing.T) {
	args := console.GoTestArgs(console.TestOptions{
		Filter:   "TestLogin",
		Coverage: true,
		Race:     true,
		Extra:    []string{"-timeout=60s"},
	})

	joined := strings.Join(args, " ")
	want := "test -json -race -cover -coverpkg=./... -count=1 -run TestLogin -timeout=60s ./..."

	if joined != want {
		t.Errorf("args = %q, want %q", joined, want)
	}
}

// -h prints usage and stops; it must not shell out or report failure.
func TestTestCommandHelpIsNotAnError(t *testing.T) {
	if err := console.NewTestCommand().Handle([]string{"-h"}); err != nil {
		t.Errorf("Handle(-h) = %v, want nil", err)
	}
}

// An unknown flag must be rejected before anything is executed.
func TestTestCommandRejectsUnknownFlag(t *testing.T) {
	if err := console.NewTestCommand().Handle([]string{"--nope"}); err == nil {
		t.Error("Handle(--nope) = nil error, want an error")
	}
}

func TestTestCommandHasDescription(t *testing.T) {
	if console.NewTestCommand().Description() == "" {
		t.Error("Description() = empty, want a description")
	}
}

// The command's --db flag and the Feature suite's gate must name the same
// variable, or --db would silently fail to enable anything.
func TestDatabaseTestsEnvIsExported(t *testing.T) {
	if console.DatabaseTestsEnv != "TEST_DB" {
		t.Errorf("DatabaseTestsEnv = %q, want TEST_DB", console.DatabaseTestsEnv)
	}
}
