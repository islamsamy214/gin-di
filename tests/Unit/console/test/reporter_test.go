package unit

import (
	"strings"
	"testing"
	"time"
	"web-app/app/console"
)

// Colour is left off throughout so the assertions compare plain text.

func TestHumanizeTestName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "simple", in: "TestLogin", want: "login"},
		{name: "camel case", in: "TestGenerateTokenParseTokenRoundTrip", want: "generate token parse token round trip"},
		{name: "acronym stays whole", in: "TestConfiguredTTLReachesToken", want: "configured ttl reaches token"},
		{name: "leading acronym", in: "TestJWTSecretIsSet", want: "jwt secret is set"},
		{name: "digits", in: "TestHS256IsPinned", want: "hs256 is pinned"},
		{name: "subtest underscores", in: "TestConfirm/yes_uppercase", want: "yes uppercase"},
		{name: "nested subtest", in: "TestA/b_c/d_e", want: "d e"},
		{name: "no Test prefix", in: "Benchmarks", want: "benchmarks"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := console.HumanizeTestName(tt.in); got != tt.want {
				t.Errorf("HumanizeTestName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// report drives the reporter with a sequence of events and returns its output.
func report(t *testing.T, events []console.TestEvent) string {
	t.Helper()

	var out strings.Builder

	reporter := &console.TestReporter{Out: &out}

	for _, event := range events {
		reporter.Handle(event)
	}

	reporter.Summary(time.Second)

	return out.String()
}

func TestReporterPrintsPassedTests(t *testing.T) {
	output := report(t, []console.TestEvent{
		{Action: "run", Package: "web-app/tests/Unit", Test: "TestLogin"},
		{Action: "pass", Package: "web-app/tests/Unit", Test: "TestLogin", Elapsed: 0.5},
		{Action: "pass", Package: "web-app/tests/Unit"},
	})

	for _, want := range []string{"tests/Unit", "✓ login", "0.50s", "1 passed", "(1 total)", "Duration:"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q:\n%s", want, output)
		}
	}
}

// The toolchain reports a parent only after its children, but starts it first,
// so run order is what keeps the nesting right.
func TestReporterPutsParentBeforeSubtests(t *testing.T) {
	output := report(t, []console.TestEvent{
		{Action: "run", Package: "web-app/tests/Unit", Test: "TestParent"},
		{Action: "run", Package: "web-app/tests/Unit", Test: "TestParent/first_case"},
		{Action: "pass", Package: "web-app/tests/Unit", Test: "TestParent/first_case"},
		{Action: "run", Package: "web-app/tests/Unit", Test: "TestParent/second_case"},
		{Action: "pass", Package: "web-app/tests/Unit", Test: "TestParent/second_case"},
		{Action: "pass", Package: "web-app/tests/Unit", Test: "TestParent"},
		{Action: "pass", Package: "web-app/tests/Unit"},
	})

	parent := strings.Index(output, "✓ parent")
	first := strings.Index(output, "first case")
	second := strings.Index(output, "second case")

	if parent == -1 || first == -1 || second == -1 {
		t.Fatalf("output missing entries:\n%s", output)
	}

	if !(parent < first && first < second) {
		t.Errorf("wrong order, want parent then subtests in run order:\n%s", output)
	}
}

func TestReporterIndentsSubtests(t *testing.T) {
	output := report(t, []console.TestEvent{
		{Action: "run", Package: "web-app/tests/Unit", Test: "TestParent"},
		{Action: "run", Package: "web-app/tests/Unit", Test: "TestParent/child"},
		{Action: "pass", Package: "web-app/tests/Unit", Test: "TestParent/child"},
		{Action: "pass", Package: "web-app/tests/Unit", Test: "TestParent"},
		{Action: "pass", Package: "web-app/tests/Unit"},
	})

	if !strings.Contains(output, "  ✓ parent\n") {
		t.Errorf("parent not indented by one level:\n%s", output)
	}

	if !strings.Contains(output, "    ✓ child\n") {
		t.Errorf("subtest not indented by two levels:\n%s", output)
	}
}

// A package's results must appear together, even when packages interleave.
func TestReporterGroupsEachPackageOnce(t *testing.T) {
	output := report(t, []console.TestEvent{
		{Action: "run", Package: "web-app/tests/Unit", Test: "TestOne"},
		{Action: "run", Package: "web-app/tests/Feature", Test: "TestTwo"},
		{Action: "pass", Package: "web-app/tests/Unit", Test: "TestOne"},
		{Action: "pass", Package: "web-app/tests/Feature", Test: "TestTwo"},
		{Action: "pass", Package: "web-app/tests/Feature"},
		{Action: "pass", Package: "web-app/tests/Unit"},
	})

	if count := strings.Count(output, "tests/Unit\n"); count != 1 {
		t.Errorf("tests/Unit header appears %d times, want 1:\n%s", count, output)
	}

	if count := strings.Count(output, "tests/Feature\n"); count != 1 {
		t.Errorf("tests/Feature header appears %d times, want 1:\n%s", count, output)
	}
}

func TestReporterCountsSkips(t *testing.T) {
	output := report(t, []console.TestEvent{
		{Action: "run", Package: "web-app/tests/Feature", Test: "TestNeedsDatabase"},
		{Action: "skip", Package: "web-app/tests/Feature", Test: "TestNeedsDatabase"},
		{Action: "pass", Package: "web-app/tests/Feature"},
	})

	if !strings.Contains(output, "↓ needs database") {
		t.Errorf("output missing the skip symbol:\n%s", output)
	}

	if !strings.Contains(output, "1 skipped") {
		t.Errorf("output missing the skip count:\n%s", output)
	}
}

// A package with no test files is a package-level skip, not a skipped test.
func TestReporterCountsPackagesWithoutTests(t *testing.T) {
	output := report(t, []console.TestEvent{
		{Action: "skip", Package: "web-app/app/models"},
		{Action: "skip", Package: "web-app/configs"},
	})

	if !strings.Contains(output, "2 packages contain no tests") {
		t.Errorf("output missing the no-tests count:\n%s", output)
	}

	if strings.Contains(output, "skipped") {
		t.Errorf("packages without tests must not count as skipped tests:\n%s", output)
	}
}

func TestReporterReportsFailuresWithOutput(t *testing.T) {
	var out strings.Builder

	reporter := &console.TestReporter{Out: &out}

	for _, event := range []console.TestEvent{
		{Action: "run", Package: "web-app/tests/Unit", Test: "TestBroken"},
		{Action: "output", Package: "web-app/tests/Unit", Test: "TestBroken", Output: "=== RUN   TestBroken\n"},
		{Action: "output", Package: "web-app/tests/Unit", Test: "TestBroken", Output: "    some_test.go:12: status = 500, want 200\n"},
		{Action: "fail", Package: "web-app/tests/Unit", Test: "TestBroken"},
		{Action: "fail", Package: "web-app/tests/Unit"},
	} {
		reporter.Handle(event)
	}

	passed := reporter.Summary(time.Second)

	if passed {
		t.Error("Summary() = true, want false when a test failed")
	}

	output := out.String()

	for _, want := range []string{"⨯ broken", "Failures", "status = 500, want 200", "1 failed"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q:\n%s", want, output)
		}
	}

	// The toolchain's own framing adds nothing to the report.
	if strings.Contains(output, "=== RUN") {
		t.Errorf("output should not repeat the toolchain framing:\n%s", output)
	}
}

func TestReporterSummaryReturnsTrueWhenAllPass(t *testing.T) {
	var out strings.Builder

	reporter := &console.TestReporter{Out: &out}
	reporter.Handle(console.TestEvent{Action: "run", Package: "web-app/tests/Unit", Test: "TestOne"})
	reporter.Handle(console.TestEvent{Action: "pass", Package: "web-app/tests/Unit", Test: "TestOne"})
	reporter.Handle(console.TestEvent{Action: "pass", Package: "web-app/tests/Unit"})

	if !reporter.Summary(time.Second) {
		t.Error("Summary() = false, want true when everything passed")
	}
}

// A package that never reports a result must not swallow its tests.
func TestReporterFlushesUnfinishedPackagesOnSummary(t *testing.T) {
	output := report(t, []console.TestEvent{
		{Action: "run", Package: "web-app/tests/Unit", Test: "TestOrphan"},
		{Action: "pass", Package: "web-app/tests/Unit", Test: "TestOrphan"},
	})

	if !strings.Contains(output, "✓ orphan") {
		t.Errorf("output lost a test whose package never finished:\n%s", output)
	}
}

func TestReporterHandlesNoTestsAtAll(t *testing.T) {
	output := report(t, nil)

	if !strings.Contains(output, "no tests ran") {
		t.Errorf("output missing the empty-run message:\n%s", output)
	}
}

// Colour must stay out of piped output.
func TestReporterOmitsColourWhenDisabled(t *testing.T) {
	output := report(t, []console.TestEvent{
		{Action: "run", Package: "web-app/tests/Unit", Test: "TestOne"},
		{Action: "pass", Package: "web-app/tests/Unit", Test: "TestOne"},
		{Action: "pass", Package: "web-app/tests/Unit"},
	})

	if strings.Contains(output, "\033[") {
		t.Errorf("output contains ANSI codes with colour disabled:\n%q", output)
	}
}

func TestReporterEmitsColourWhenEnabled(t *testing.T) {
	var out strings.Builder

	reporter := &console.TestReporter{Out: &out, Color: true}
	reporter.Handle(console.TestEvent{Action: "run", Package: "web-app/tests/Unit", Test: "TestOne"})
	reporter.Handle(console.TestEvent{Action: "pass", Package: "web-app/tests/Unit", Test: "TestOne"})
	reporter.Handle(console.TestEvent{Action: "pass", Package: "web-app/tests/Unit"})
	reporter.Summary(time.Second)

	if !strings.Contains(out.String(), "\033[") {
		t.Errorf("output has no ANSI codes with colour enabled:\n%q", out.String())
	}
}
