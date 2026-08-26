package core

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"web-app/configs"
)

// Log storage layout and retention.
const (
	logDirectory     = "./storage/logs"
	logDirectoryMode = 0o755
	logFileMode      = 0o644
	logFileExtension = ".log"
	logDateLayout    = "2006-01-02"
	logRetentionDays = 30
)

// localEnv is the one environment that gets human-readable output; everywhere
// else logs are JSON so they can be queried rather than grepped.
const localEnv = "local"

/*
 * NewLogger builds the application logger and the file behind it.
 *
 * @param appConfig Supplies the log level, the environment, and the file name.
 * @return *slog.Logger The logger.
 * @return io.Closer    The log file, which the composition root must close on
 *                      shutdown. The previous implementation held the handle
 *                      open for the lifetime of the process with no way to
 *                      release it.
 * @return error        If the log directory or file could not be prepared.
 */
func NewLogger(appConfig *configs.AppConfig) (*slog.Logger, io.Closer, error) {
	writer, err := newDailyWriter(logDirectory, appConfig.Name)
	if err != nil {
		return nil, nil, err
	}

	// Both destinations on purpose: stdout is what a container orchestrator
	// collects, the file is what survives a crash loop long enough to read.
	options := &slog.HandlerOptions{Level: parseLevel(appConfig.LogLevel)}
	output := io.MultiWriter(os.Stdout, writer)

	var handler slog.Handler = slog.NewJSONHandler(output, options)
	if appConfig.Env == localEnv {
		handler = slog.NewTextHandler(output, options)
	}

	return slog.New(handler), writer, nil
}

/*
 * parseLevel maps a configured level name onto a slog level.
 *
 * An unrecognised name yields Info rather than an error: a typo in APP_LOG_LEVEL
 * should not stop the application from booting, and Info is the safe direction
 * to fail in.
 */
func parseLevel(name string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

/*
 * dailyWriter appends to a dated log file, switching files when the date rolls
 * over.
 *
 * The file name used to be computed once at construction, so a process running
 * past midnight kept writing yesterday's file forever. Here the date is
 * re-checked on every write, which is the only point at which the writer is
 * guaranteed to be running.
 */
type dailyWriter struct {
	directory string
	prefix    string

	mutex sync.Mutex
	file  *os.File
	date  string
}

/*
 * newDailyWriter prepares the log directory and opens today's file.
 *
 * @param directory Where log files live.
 * @param prefix    The application name the file is named after.
 */
func newDailyWriter(directory, prefix string) (*dailyWriter, error) {
	// Checked, and 0o755 rather than the os.ModePerm (0777) it used to be: a
	// world-writable log directory lets any local user forge audit records.
	if err := os.MkdirAll(directory, logDirectoryMode); err != nil {
		return nil, fmt.Errorf("preparing log directory %s: %w", directory, err)
	}

	writer := &dailyWriter{directory: directory, prefix: prefix}

	if err := writer.rotate(time.Now().Format(logDateLayout)); err != nil {
		return nil, err
	}

	return writer, nil
}

// Write appends to today's file, rolling over first if the date has changed.
func (writer *dailyWriter) Write(record []byte) (int, error) {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()

	if today := time.Now().Format(logDateLayout); today != writer.date {
		if err := writer.rotate(today); err != nil {
			return 0, err
		}
	}

	return writer.file.Write(record)
}

// Close releases the current file.
func (writer *dailyWriter) Close() error {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()

	if writer.file == nil {
		return nil
	}

	file := writer.file
	writer.file = nil

	return file.Close()
}

/*
 * rotate switches to the file for the given date and prunes expired ones.
 *
 * Callers must hold the mutex. The retention sweep runs here — synchronously,
 * with its error surfaced — rather than in a detached goroutine whose failures
 * nothing could observe.
 */
func (writer *dailyWriter) rotate(date string) error {
	name := filepath.Join(writer.directory, writer.prefix+"-"+date+logFileExtension)

	// nolint:gosec // G304: the path is built from a constant directory and the
	// configured application name, never from request input.
	file, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, logFileMode)
	if err != nil {
		return fmt.Errorf("opening log file %s: %w", name, err)
	}

	if writer.file != nil {
		// The new file is already open, so a failure to close the old one is
		// worth recording but must not take the logger down with it.
		if closeErr := writer.file.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "closing previous log file: %v\n", closeErr)
		}
	}

	writer.file = file
	writer.date = date

	if err := pruneLogs(writer.directory, logRetentionDays); err != nil {
		fmt.Fprintf(os.Stderr, "pruning old logs: %v\n", err)
	}

	return nil
}

/*
 * pruneLogs removes log files last modified before the retention cutoff.
 *
 * Only files carrying the log extension are considered: the old sweep matched
 * every directory entry by modification time, so it would have deleted the
 * .gitkeep that keeps storage/logs in the repository.
 *
 * @param directory     The directory to sweep.
 * @param retentionDays How many days of logs to keep.
 * @return error The first deletion failure, or nil.
 */
func pruneLogs(directory string, retentionDays int) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("reading log directory %s: %w", directory, err)
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != logFileExtension {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("inspecting %s: %w", entry.Name(), err)
		}

		if info.ModTime().After(cutoff) {
			continue
		}

		if err := os.Remove(filepath.Join(directory, entry.Name())); err != nil {
			return fmt.Errorf("removing %s: %w", entry.Name(), err)
		}
	}

	return nil
}
