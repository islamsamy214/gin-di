package test

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"
)

// Status symbols, matching the shape of Pest's output.
const (
	passSymbol = "✓"
	failSymbol = "⨯"
	skipSymbol = "↓"
)

// ANSI colours, emitted only when the destination is a terminal.
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorDim    = "\033[2m"
	colorBold   = "\033[1m"
)

/*
 * Event is one record from `go test -json`.
 *
 * Package-level records carry an empty Test, which is how a package result is
 * told apart from an individual test result.
 */
type Event struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
	Output  string  `json:"Output"`
}

// testResult is one test's outcome plus whatever it printed.
type testResult struct {
	name    string
	status  string
	elapsed float64
	output  []string
}

/*
 * packageResults collects one package's tests.
 *
 * order preserves the sequence the tests started in, which is what puts a
 * parent above its subtests: the toolchain reports a parent as finished only
 * after its children, but starts it before them.
 */
type packageResults struct {
	order   []string
	results map[string]*testResult
}

/*
 * Reporter renders `go test -json` events as a Laravel-style report.
 *
 * Results are held until their package finishes, then printed together. The
 * toolchain interleaves packages and reports children before parents, so
 * printing on arrival would scatter one package across the output and invert
 * the nesting.
 */
type Reporter struct {
	Out   io.Writer
	Color bool

	passed  int
	failed  int
	skipped int
	noTests int

	packages map[string]*packageResults
	failures []testResult
	failedIn map[string]string
}

/*
 * Handle consumes one event, printing a package's report once it completes.
 */
func (reporter *Reporter) Handle(event Event) {
	if reporter.packages == nil {
		reporter.packages = make(map[string]*packageResults)
		reporter.failedIn = make(map[string]string)
	}

	if event.Test == "" {
		reporter.handlePackage(event)

		return
	}

	pkg, found := reporter.packages[event.Package]
	if !found {
		pkg = &packageResults{results: make(map[string]*testResult)}
		reporter.packages[event.Package] = pkg
	}

	result, found := pkg.results[event.Test]
	if !found {
		result = &testResult{name: event.Test}
		pkg.results[event.Test] = result
		pkg.order = append(pkg.order, event.Test)
	}

	switch event.Action {
	case "output":
		result.output = append(result.output, event.Output)
	case "pass", "fail", "skip":
		result.status = event.Action
		result.elapsed = event.Elapsed
	}
}

/*
 * handlePackage flushes a finished package, or counts one that had no tests.
 */
func (reporter *Reporter) handlePackage(event Event) {
	switch event.Action {
	case "skip":
		// A package with no test files is reported as a skipped package.
		reporter.noTests++
	case "pass", "fail":
		reporter.flush(event.Package)
	}
}

/*
 * flush prints one package's results in the order its tests started.
 */
func (reporter *Reporter) flush(name string) {
	pkg, found := reporter.packages[name]
	if !found {
		return
	}

	delete(reporter.packages, name)

	if len(pkg.order) == 0 {
		return
	}

	fmt.Fprintf(reporter.Out, "\n%s\n", reporter.paint(shortPackageName(name), colorBold))

	for _, testName := range pkg.order {
		result := pkg.results[testName]

		switch result.status {
		case "pass":
			reporter.passed++
			reporter.printResult(passSymbol, colorGreen, result)
		case "fail":
			reporter.failed++
			reporter.printResult(failSymbol, colorRed, result)
			reporter.failures = append(reporter.failures, *result)
			reporter.failedIn[result.name] = name
		case "skip":
			reporter.skipped++
			reporter.printResult(skipSymbol, colorYellow, result)
		}
	}
}

/*
 * printResult writes one result line, indented by its subtest depth.
 */
func (reporter *Reporter) printResult(symbol, color string, result *testResult) {
	indent := strings.Repeat("  ", strings.Count(result.name, "/")+1)

	line := fmt.Sprintf("%s%s %s", indent, reporter.paint(symbol, color), HumanizeTestName(result.name))

	// Only surface a duration when it is long enough to be worth noticing.
	if result.elapsed >= 0.01 {
		line += reporter.paint(fmt.Sprintf("  %.2fs", result.elapsed), colorDim)
	}

	fmt.Fprintln(reporter.Out, line)
}

/*
 * Summary prints the failure details and the totals.
 *
 * @return bool Whether every test passed.
 */
func (reporter *Reporter) Summary(elapsed time.Duration) bool {
	// Any package still held here never reported a result, so flush it rather
	// than losing the tests silently.
	for name := range reporter.packages {
		reporter.flush(name)
	}

	reporter.printFailures()

	counts := make([]string, 0, 3)

	if reporter.failed > 0 {
		counts = append(counts, reporter.paint(fmt.Sprintf("%d failed", reporter.failed), colorRed))
	}

	if reporter.skipped > 0 {
		counts = append(counts, reporter.paint(fmt.Sprintf("%d skipped", reporter.skipped), colorYellow))
	}

	if reporter.passed > 0 {
		counts = append(counts, reporter.paint(fmt.Sprintf("%d passed", reporter.passed), colorGreen))
	}

	total := reporter.passed + reporter.failed + reporter.skipped

	if total == 0 {
		counts = append(counts, "no tests ran")
	}

	fmt.Fprintf(reporter.Out, "\n  %s  %s (%d total)\n",
		reporter.paint("Tests:", colorBold), strings.Join(counts, ", "), total)
	fmt.Fprintf(reporter.Out, "  %s  %.2fs\n",
		reporter.paint("Duration:", colorBold), elapsed.Seconds())

	if reporter.noTests > 0 {
		fmt.Fprintf(reporter.Out, "  %s\n",
			reporter.paint(fmt.Sprintf("%d packages contain no tests", reporter.noTests), colorDim))
	}

	return reporter.failed == 0
}

/*
 * printFailures repeats each failure with its output, so a long run does not
 * require scrolling back to find what broke.
 */
func (reporter *Reporter) printFailures() {
	if len(reporter.failures) == 0 {
		return
	}

	fmt.Fprintf(reporter.Out, "\n%s\n", reporter.paint("Failures", colorBold))

	for _, failed := range reporter.failures {
		fmt.Fprintf(reporter.Out, "\n  %s %s %s %s\n",
			reporter.paint(failSymbol, colorRed),
			shortPackageName(reporter.failedIn[failed.name]),
			reporter.paint("›", colorDim),
			HumanizeTestName(failed.name),
		)

		for _, line := range failed.output {
			trimmed := strings.TrimSpace(line)

			// The toolchain's own framing adds nothing here.
			if trimmed == "" || strings.HasPrefix(trimmed, "--- FAIL") || strings.HasPrefix(trimmed, "=== RUN") {
				continue
			}

			fmt.Fprintf(reporter.Out, "    %s\n", reporter.paint(trimmed, colorDim))
		}
	}
}

// paint wraps text in an ANSI colour when the output is a terminal.
func (reporter *Reporter) paint(text, color string) string {
	if !reporter.Color {
		return text
	}

	return color + text + colorReset
}

/*
 * HumanizeTestName turns a Go test identifier into a readable phrase.
 *
 * TestGenerateToken becomes "generate token". A subtest's leaf is used as
 * written, since t.Run already turned its spaces into underscores.
 *
 * @return string The readable name.
 */
func HumanizeTestName(name string) string {
	parts := strings.Split(name, "/")
	leaf := parts[len(parts)-1]

	if len(parts) > 1 {
		return strings.ReplaceAll(leaf, "_", " ")
	}

	runes := []rune(strings.TrimPrefix(leaf, "Test"))

	var builder strings.Builder

	for index, current := range runes {
		if index > 0 && unicode.IsUpper(current) {
			previousEndsWord := unicode.IsLower(runes[index-1]) || unicode.IsDigit(runes[index-1])
			nextStartsWord := index+1 < len(runes) && unicode.IsLower(runes[index+1])

			// Only break on a case change, so an acronym stays one word.
			if previousEndsWord || nextStartsWord {
				builder.WriteRune(' ')
			}
		}

		builder.WriteRune(unicode.ToLower(current))
	}

	return builder.String()
}

/*
 * shortPackageName drops the module prefix from an import path.
 *
 * @return string The path relative to the module, or the path itself.
 */
func shortPackageName(pkg string) string {
	if index := strings.Index(pkg, "/"); index != -1 {
		return pkg[index+1:]
	}

	return pkg
}

/*
 * IsTerminal reports whether the file is attached to a terminal.
 *
 * Used to keep ANSI codes out of piped or redirected output.
 *
 * @return bool Whether colour is safe to emit.
 */
func IsTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}
