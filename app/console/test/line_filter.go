package test

import (
	"bytes"
	"io"
	"strings"
)

// noTestFilesMarker is what the toolchain prints for a package without tests.
const noTestFilesMarker = "[no test files]"

/*
 * LineFilter passes output through to Out, dropping whole lines that Drop
 * matches, and counting how many it removed.
 *
 * Buffers until a newline arrives, because a single Write is not guaranteed to
 * contain whole lines: filtering per Write would corrupt output split mid-line.
 */
type LineFilter struct {
	Out     io.Writer
	Drop    func(line string) bool
	Dropped int

	pending []byte
}

/*
 * Write buffers the given bytes and forwards every complete, unfiltered line.
 *
 * @return int   Always len(p): every byte is consumed, filtered or forwarded.
 * @return error If the underlying writer failed.
 */
func (filter *LineFilter) Write(p []byte) (int, error) {
	filter.pending = append(filter.pending, p...)

	for {
		index := bytes.IndexByte(filter.pending, '\n')
		if index < 0 {
			break
		}

		line := filter.pending[:index+1]
		filter.pending = filter.pending[index+1:]

		if filter.Drop != nil && filter.Drop(string(line)) {
			filter.Dropped++

			continue
		}

		if _, err := filter.Out.Write(line); err != nil {
			return 0, err
		}
	}

	return len(p), nil
}

/*
 * Flush forwards a trailing line that never ended in a newline.
 *
 * @return error If the underlying writer failed.
 */
func (filter *LineFilter) Flush() error {
	if len(filter.pending) == 0 {
		return nil
	}

	line := filter.pending
	filter.pending = nil

	if filter.Drop != nil && filter.Drop(string(line)) {
		filter.Dropped++

		return nil
	}

	_, err := filter.Out.Write(line)

	return err
}

/*
 * IsNoTestFilesLine reports whether a line is the toolchain's "this package has
 * no tests" notice.
 *
 * With the whole suite living under tests/, these lines outnumber the results
 * and bury them.
 *
 * @return bool Whether the line carries no information worth printing.
 */
func IsNoTestFilesLine(line string) bool {
	return strings.Contains(line, noTestFilesMarker)
}
