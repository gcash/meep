package main

import (
	"bufio"
	"bytes"
	"testing"

	tm "github.com/buger/goterm"
)

func TestFitFrame(t *testing.T) {
	tests := []struct {
		name          string
		frame         string
		width, height int
		want          string
	}{
		{
			name:  "fits",
			frame: "a\nb\n",
			width: 10, height: 5,
			want: "a\nb",
		},
		{
			name:  "too many lines",
			frame: "a\nb\nc\n",
			width: 10, height: 2,
			want: "a\nb",
		},
		{
			// The first line wraps onto 3 rows.
			name:  "long line",
			frame: "aaaaaaaaaaaa\nb\n",
			width: 5, height: 3,
			want: "aaaaaaaaaaaa",
		},
		{
			name:  "escape sequences take no room",
			frame: "\x1b[1mabc\x1b[0m\nd\n",
			width: 3, height: 2,
			want: "\x1b[1mabc\x1b[0m\nd",
		},
		{
			name:  "unknown size",
			frame: "a\nb\n",
			width: -1, height: -1,
			want: "a\nb",
		},
	}
	for _, test := range tests {
		if got := fitFrame(test.frame, test.width, test.height); got != test.want {
			t.Errorf("%s: got %q, want %q", test.name, got, test.want)
		}
	}
}

// TestFlushDrawsEveryFrame checks that each frame is drawn and the buffer is
// emptied, so the display never freezes on an old frame.
func TestFlushDrawsEveryFrame(t *testing.T) {
	var out bytes.Buffer
	saved := tm.Output
	tm.Output = bufio.NewWriter(&out)
	defer func() { tm.Output = saved }()

	tm.Screen.Reset()
	_, _ = tm.Println("frame 1")
	flush()
	_, _ = tm.Println("frame 2")
	flush()
	if got, want := out.String(), "frame 1frame 2"; got != want {
		t.Errorf("drew %q, want %q", got, want)
	}
	if tm.Screen.Len() != 0 {
		t.Errorf("buffer still holds %q", tm.Screen.String())
	}
}
