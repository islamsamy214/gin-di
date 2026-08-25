package unit

import (
	"errors"
	"io"
	"strings"
	"testing"
	"web-app/app/console"
)

func TestConfirmAcceptsOnlyYes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "yes", input: "yes\n", want: true},
		{name: "yes uppercase", input: "YES\n", want: true},
		{name: "yes mixed case", input: "Yes\n", want: true},
		{name: "yes with surrounding spaces", input: "  yes  \n", want: true},
		{name: "yes without a newline", input: "yes", want: true},

		// Anything short of the whole word must not destroy data.
		{name: "y alone", input: "y\n", want: false},
		{name: "no", input: "no\n", want: false},
		{name: "empty line", input: "\n", want: false},
		{name: "closed input", input: "", want: false},
		{name: "yeah", input: "yeah\n", want: false},
		{name: "yes plus text", input: "yes please\n", want: false},
		{name: "unrelated", input: "drop it\n", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder

			got, err := console.Confirm(strings.NewReader(tt.input), &out, "Continue?")
			if err != nil {
				t.Fatalf("Confirm() = %v, want nil", err)
			}

			if got != tt.want {
				t.Errorf("Confirm(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// The operator must see what they are approving before answering.
func TestConfirmWritesThePrompt(t *testing.T) {
	const prompt = "This DROPS EVERY TABLE in homestead. Continue?"

	var out strings.Builder

	if _, err := console.Confirm(strings.NewReader("no\n"), &out, prompt); err != nil {
		t.Fatalf("Confirm() = %v, want nil", err)
	}

	written := out.String()

	if !strings.Contains(written, prompt) {
		t.Errorf("prompt = %q, want it to contain %q", written, prompt)
	}

	if !strings.Contains(written, "yes") {
		t.Errorf("prompt = %q, want it to state the required answer", written)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("no tty")
}

// If the prompt cannot be shown, treat it as an error rather than proceeding on
// an answer nobody was asked for.
func TestConfirmFailsWhenPromptCannotBeWritten(t *testing.T) {
	got, err := console.Confirm(strings.NewReader("yes\n"), failingWriter{}, "Continue?")
	if err == nil {
		t.Error("Confirm() = nil error, want an error")
	}

	if got {
		t.Error("Confirm() = true, want false when the prompt could not be shown")
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestConfirmFailsWhenInputCannotBeRead(t *testing.T) {
	var out strings.Builder

	got, err := console.Confirm(failingReader{}, &out, "Continue?")
	if err == nil {
		t.Error("Confirm() = nil error, want an error")
	}

	if got {
		t.Error("Confirm() = true, want false when input could not be read")
	}
}

// A closed stdin is EOF, not a failure — it simply means "not confirmed".
func TestConfirmTreatsEOFAsDeclined(t *testing.T) {
	var out strings.Builder

	got, err := console.Confirm(io.LimitReader(strings.NewReader(""), 0), &out, "Continue?")
	if err != nil {
		t.Fatalf("Confirm() = %v, want nil", err)
	}

	if got {
		t.Error("Confirm() = true, want false on EOF")
	}
}
