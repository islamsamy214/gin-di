package console

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// confirmationWord is the exact answer required to proceed. A whole word rather
// than "y" so a destructive command cannot be approved by a stray keystroke.
const confirmationWord = "yes"

/*
 * Confirm asks the operator to approve a destructive change.
 *
 * Takes the reader and writer as arguments rather than reaching for os.Stdin,
 * so callers can pipe input and tests can drive it directly.
 *
 * @return bool  Whether the operator typed the confirmation word.
 * @return error If the answer could not be read.
 */
func Confirm(in io.Reader, out io.Writer, prompt string) (bool, error) {
	if _, err := fmt.Fprintf(out, "%s [%s/no]: ", prompt, confirmationWord); err != nil {
		return false, fmt.Errorf("writing prompt: %w", err)
	}

	answer, err := bufio.NewReader(in).ReadString('\n')

	// A closed stdin still yields whatever was typed before EOF.
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("reading confirmation: %w", err)
	}

	return strings.EqualFold(strings.TrimSpace(answer), confirmationWord), nil
}
