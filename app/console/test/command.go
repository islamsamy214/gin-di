package test

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

const (
	// testPackages is the pattern handed to the toolchain; ./... is the module.
	testPackages = "./..."

	// DatabaseTestsEnv gates the tests that drop tables. The Feature suite
	// skips unless this is set, so --db exists to opt in from here.
	DatabaseTestsEnv = "TEST_DB"

	// outputBufferLimit caps one line of toolchain output. Failure messages can
	// be long, and the default scanner limit would truncate them.
	outputBufferLimit = 1024 * 1024
)

/*
 * Options describes one test run.
 *
 * Separated from the flag set so the arguments can be built and asserted on
 * without executing anything.
 */
type Options struct {
	Filter   string
	Coverage bool
	Race     bool
	Cache    bool
	Raw      bool
	Database bool
	Extra    []string
}

type Command struct{}

func NewCommand() *Command {
	return &Command{}
}

/*
 * Handle runs the test suite, the way `php artisan test` fronts PHPUnit.
 *
 * Every test is reported as it finishes, followed by a summary. A failing
 * suite surfaces as a non-zero exit.
 */
func (command *Command) Handle(args []string) error {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	filter := flags.String("filter", "", "run only tests whose name matches this regular expression")
	coverage := flags.Bool("coverage", false, "report statement coverage per package")
	race := flags.Bool("race", false, "enable the race detector")
	cache := flags.Bool("cache", false, "allow cached results instead of re-running everything")
	raw := flags.Bool("raw", false, "print the toolchain's own output instead of the formatted report")
	withDatabase := flags.Bool("db", false, "also run the database tests by setting "+DatabaseTestsEnv)

	if err := flags.Parse(args); err != nil {
		// -h already printed usage; that is not a failure.
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return fmt.Errorf("parsing flags: %w", err)
	}

	options := Options{
		Filter:   *filter,
		Coverage: *coverage,
		Race:     *race,
		Cache:    *cache,
		Raw:      *raw,
		Database: *withDatabase,
		Extra:    flags.Args(),
	}

	goBinary, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("finding the go toolchain: %w", err)
	}

	if !*withDatabase {
		log.Printf("database tests will be skipped, pass --db to include them")
	}

	suite := exec.Command(goBinary, GoTestArgs(options)...)
	suite.Stderr = os.Stderr
	suite.Stdin = os.Stdin
	suite.Env = os.Environ()

	if *withDatabase {
		suite.Env = append(suite.Env, DatabaseTestsEnv+"=1")
	}

	if options.Raw {
		return command.runRaw(suite)
	}

	return command.runReported(suite)
}

func (command *Command) Description() string {
	return "Runs the test suite (--filter, --coverage, --race, --db, --raw)"
}

/*
 * runReported streams `go test -json` through the reporter.
 *
 * @return error If the suite failed, or its output could not be read.
 */
func (command *Command) runReported(suite *exec.Cmd) error {
	stdout, err := suite.StdoutPipe()
	if err != nil {
		return fmt.Errorf("capturing test output: %w", err)
	}

	startedAt := time.Now()

	if err := suite.Start(); err != nil {
		return fmt.Errorf("starting the test suite: %w", err)
	}

	reporter := &Reporter{Out: os.Stdout, Color: IsTerminal(os.Stdout)}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), outputBufferLimit)

	for scanner.Scan() {
		line := scanner.Bytes()

		var event Event

		// Build failures and toolchain notices are not JSON records, so they
		// are passed through rather than swallowed.
		if err := json.Unmarshal(line, &event); err != nil {
			fmt.Fprintln(os.Stdout, string(line))

			continue
		}

		reporter.Handle(event)
	}

	scanErr := scanner.Err()
	runErr := suite.Wait()

	passed := reporter.Summary(time.Since(startedAt))

	if scanErr != nil {
		return fmt.Errorf("reading test output: %w", scanErr)
	}

	if runErr != nil {
		var exitErr *exec.ExitError

		// The report already showed what failed, so keep this short.
		if errors.As(runErr, &exitErr) {
			return errors.New("test suite failed")
		}

		return fmt.Errorf("running the test suite: %w", runErr)
	}

	if !passed {
		return errors.New("test suite failed")
	}

	return nil
}

/*
 * runRaw forwards the toolchain's own output, dropping the "no test files"
 * notices that would otherwise bury the results.
 *
 * @return error If the suite failed.
 */
func (command *Command) runRaw(suite *exec.Cmd) error {
	stdout := &LineFilter{Out: os.Stdout, Drop: IsNoTestFilesLine}
	suite.Stdout = stdout

	runErr := suite.Run()

	if err := stdout.Flush(); err != nil {
		return fmt.Errorf("writing test output: %w", err)
	}

	if stdout.Dropped > 0 {
		log.Printf("%d packages contain no tests", stdout.Dropped)
	}

	if runErr != nil {
		var exitErr *exec.ExitError

		if errors.As(runErr, &exitErr) {
			return errors.New("test suite failed")
		}

		return fmt.Errorf("running the test suite: %w", runErr)
	}

	return nil
}

/*
 * GoTestArgs builds the argument list for the toolchain.
 *
 * Pure on purpose: the command shells out to `go test`, so this is the only
 * part a test can exercise without invoking itself.
 *
 * @return []string The full argument list, starting with "test".
 */
func GoTestArgs(options Options) []string {
	args := []string{"test"}

	// -json carries one record per test, which is what lets every result be
	// reported without scraping text. It implies verbose output.
	if options.Raw {
		args = append(args, "-v")
	} else {
		args = append(args, "-json")
	}

	if options.Race {
		args = append(args, "-race")
	}

	// -coverpkg is required, not cosmetic: the tests live in their own
	// packages under tests/, so plain -cover would attribute nothing to the
	// application packages and report 0% everywhere.
	if options.Coverage {
		args = append(args, "-cover", "-coverpkg="+testPackages)
	}

	// Cached results make a "run my tests" command lie, most of all for the
	// database tests whose outcome depends on state the cache cannot see.
	if !options.Cache {
		args = append(args, "-count=1")
	}

	// The database tests share one schema and every FreshDatabase call drops
	// every table, so two packages running at once would tear down each other's
	// fixtures. -p 1 runs one test binary at a time; only database runs pay for
	// it, because only they touch shared state.
	if options.Database {
		args = append(args, "-p", "1")
	}

	if options.Filter != "" {
		args = append(args, "-run", options.Filter)
	}

	// Anything after -- goes straight through to the toolchain.
	args = append(args, options.Extra...)

	return append(args, testPackages)
}
