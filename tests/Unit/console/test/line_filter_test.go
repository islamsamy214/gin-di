package unit

import (
	"errors"
	"strings"
	"testing"
	"web-app/app/console/test"
)

func TestIsNoTestFilesLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{name: "toolchain notice", line: "?   \tweb-app/app/console\t[no test files]\n", want: true},
		{name: "notice without newline", line: "?   \tweb-app/configs\t[no test files]", want: true},
		{name: "passing package", line: "ok  \tweb-app/tests/Unit\t0.126s\n", want: false},
		{name: "failing package", line: "FAIL\tweb-app/tests/Unit\t0.005s\n", want: false},
		{name: "test result", line: "--- PASS: TestLogin (0.00s)\n", want: false},
		{name: "coverage line", line: "ok  \tweb-app/tests/Unit\t0.1s\tcoverage: 22.5% of statements\n", want: false},
		{name: "empty", line: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := test.IsNoTestFilesLine(tt.line); got != tt.want {
				t.Errorf("IsNoTestFilesLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestLineFilterDropsMatchingLines(t *testing.T) {
	var out strings.Builder

	filter := &test.LineFilter{Out: &out, Drop: test.IsNoTestFilesLine}

	input := "?   \tweb-app\t[no test files]\n" +
		"ok  \tweb-app/tests/Unit\t0.126s\n" +
		"?   \tweb-app/configs\t[no test files]\n" +
		"ok  \tweb-app/tests/Feature\t0.007s\n"

	if _, err := filter.Write([]byte(input)); err != nil {
		t.Fatalf("Write() = %v, want nil", err)
	}

	if err := filter.Flush(); err != nil {
		t.Fatalf("Flush() = %v, want nil", err)
	}

	want := "ok  \tweb-app/tests/Unit\t0.126s\n" +
		"ok  \tweb-app/tests/Feature\t0.007s\n"

	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}

	if filter.Dropped != 2 {
		t.Errorf("Dropped = %d, want 2", filter.Dropped)
	}
}

// A Write is not guaranteed to contain whole lines, so filtering per Write
// would corrupt output that arrives split mid-line.
func TestLineFilterHandlesLinesSplitAcrossWrites(t *testing.T) {
	var out strings.Builder

	filter := &test.LineFilter{Out: &out, Drop: test.IsNoTestFilesLine}

	// "[no test files]" straddles the boundary and must still be dropped.
	for _, chunk := range []string{"?   \tweb-app\t[no te", "st files]\nok  \tweb-app/tests/Unit\t0.1", "26s\n"} {
		if _, err := filter.Write([]byte(chunk)); err != nil {
			t.Fatalf("Write(%q) = %v, want nil", chunk, err)
		}
	}

	if err := filter.Flush(); err != nil {
		t.Fatalf("Flush() = %v, want nil", err)
	}

	const want = "ok  \tweb-app/tests/Unit\t0.126s\n"

	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}

	if filter.Dropped != 1 {
		t.Errorf("Dropped = %d, want 1", filter.Dropped)
	}
}

// A byte-at-a-time writer is the worst case for the buffering.
func TestLineFilterHandlesByteAtATimeWrites(t *testing.T) {
	var out strings.Builder

	filter := &test.LineFilter{Out: &out, Drop: test.IsNoTestFilesLine}

	input := "?   \tweb-app\t[no test files]\nFAIL\tweb-app/tests/Unit\t0.005s\n"

	for index := range len(input) {
		if _, err := filter.Write([]byte{input[index]}); err != nil {
			t.Fatalf("Write() = %v, want nil", err)
		}
	}

	if err := filter.Flush(); err != nil {
		t.Fatalf("Flush() = %v, want nil", err)
	}

	const want = "FAIL\tweb-app/tests/Unit\t0.005s\n"

	if out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}

// Output with no trailing newline must still reach the writer.
func TestLineFilterFlushesTrailingLine(t *testing.T) {
	var out strings.Builder

	filter := &test.LineFilter{Out: &out, Drop: test.IsNoTestFilesLine}

	if _, err := filter.Write([]byte("FAIL\tweb-app/tests/Unit\t0.005s")); err != nil {
		t.Fatalf("Write() = %v, want nil", err)
	}

	if out.String() != "" {
		t.Errorf("output = %q, want nothing before Flush", out.String())
	}

	if err := filter.Flush(); err != nil {
		t.Fatalf("Flush() = %v, want nil", err)
	}

	if out.String() != "FAIL\tweb-app/tests/Unit\t0.005s" {
		t.Errorf("output = %q, want the trailing line", out.String())
	}
}

// A nil Drop must pass everything through, which is what --all-packages relies on.
func TestLineFilterWithoutDropPassesEverything(t *testing.T) {
	var out strings.Builder

	filter := &test.LineFilter{Out: &out}

	input := "?   \tweb-app\t[no test files]\nok  \tweb-app/tests/Unit\t0.1s\n"

	if _, err := filter.Write([]byte(input)); err != nil {
		t.Fatalf("Write() = %v, want nil", err)
	}

	if out.String() != input {
		t.Errorf("output = %q, want %q", out.String(), input)
	}

	if filter.Dropped != 0 {
		t.Errorf("Dropped = %d, want 0", filter.Dropped)
	}
}

// Write must report every byte consumed, or callers see a short-write error.
func TestLineFilterReportsAllBytesConsumed(t *testing.T) {
	var out strings.Builder

	filter := &test.LineFilter{Out: &out, Drop: test.IsNoTestFilesLine}

	input := []byte("?   \tweb-app\t[no test files]\n")

	written, err := filter.Write(input)
	if err != nil {
		t.Fatalf("Write() = %v, want nil", err)
	}

	if written != len(input) {
		t.Errorf("Write() = %d, want %d even though the line was dropped", written, len(input))
	}
}

type failingOut struct{}

func (failingOut) Write([]byte) (int, error) {
	return 0, errors.New("broken pipe")
}

func TestLineFilterSurfacesWriteErrors(t *testing.T) {
	filter := &test.LineFilter{Out: failingOut{}, Drop: test.IsNoTestFilesLine}

	if _, err := filter.Write([]byte("ok  \tweb-app/tests/Unit\t0.1s\n")); err == nil {
		t.Error("Write() = nil error, want the underlying failure")
	}
}
