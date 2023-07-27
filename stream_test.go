package cmd_test

import (
	"testing"

	"github.com/bingoohuang/cmd"
	"github.com/go-test/deep"
)

func TestStreamingMultipleLines(t *testing.T) {
	lines := make(chan string, 5)
	out := cmd.NewOutputStream(lines)

	// Quick side test: Lines() chan string should be the same chan string
	// we created the object with
	if out.Lines() != lines {
		t.Errorf("Lines() does not return the given string chan")
	}

	// Write two short lines
	input := "foo\nbar\n"
	n, err := out.Write([]byte(input))

	if n != len(input) {
		t.Errorf("Write n = %d, expected %d", n, len(input))
	}

	if err != nil {
		t.Errorf("got err '%v', expected nil", err)
	}

	// Get one line
	var gotLine string
	select {
	case gotLine = <-lines:
	default:
		t.Fatal("blocked on <-lines")
	}

	// "foo" should be sent before "bar" because that was the input
	if gotLine != "foo" { // nolint
		t.Errorf("got line: '%s', expected 'foo'", gotLine)
	}

	// Get next line
	select {
	case gotLine = <-lines:
	default:
		t.Fatal("blocked on <-lines")
	}

	if gotLine != "bar" {
		t.Errorf("got line: '%s', expected 'bar'", gotLine)
	}
}

func TestStreamingBlankLines(t *testing.T) { // nolint funlen
	lines := make(chan string, 5)
	out := cmd.NewOutputStream(lines)

	// Blank line in the middle
	input := "foo\n\nbar\n"
	expectLines := []string{"foo", "", "bar"}
	gotLines := []string{}
	n, err := out.Write([]byte(input))

	if n != len(input) {
		t.Errorf("Write n = %d, expected %d", n, len(input))
	}

	if err != nil {
		t.Errorf("got err '%v', expected nil", err)
	}

LINES1:
	for {
		select {
		case line := <-lines:
			gotLines = append(gotLines, line)
		default:
			break LINES1
		}
	}

	if diffs := deep.Equal(gotLines, expectLines); diffs != nil {
		t.Error(diffs)
	}

	// All blank lines
	input = "\n\n\n"
	expectLines = []string{"", "", ""}
	gotLines = []string{}
	n, err = out.Write([]byte(input))

	if n != len(input) {
		t.Errorf("Write n = %d, expected %d", n, len(input))
	}

	if err != nil {
		t.Errorf("got err '%v', expected nil", err)
	}

LINES2:
	for {
		select {
		case line := <-lines:
			gotLines = append(gotLines, line)
		default:
			break LINES2
		}
	}

	if diffs := deep.Equal(gotLines, expectLines); diffs != nil {
		t.Error(diffs)
	}

	// Blank lines at end
	input = "foo\n\n\n"
	expectLines = []string{"foo", "", ""}
	gotLines = []string{}
	n, err = out.Write([]byte(input))

	if n != len(input) {
		t.Errorf("Write n = %d, expected %d", n, len(input))
	}

	if err != nil {
		t.Errorf("got err '%v', expected nil", err)
	}

LINES3:
	for {
		select {
		case line := <-lines:
			gotLines = append(gotLines, line)
		default:
			break LINES3
		}
	}

	if diffs := deep.Equal(gotLines, expectLines); diffs != nil {
		t.Error(diffs)
	}
}

func TestStreamingCarriageReturn(t *testing.T) {
	// Carriage return should be stripped
	lines := make(chan string, 5)
	out := cmd.NewOutputStream(lines)

	input := "foo\r\nbar\r\n"
	expectLines := []string{"foo", "bar"}

	var gotLines []string

	n, err := out.Write([]byte(input))

	if n != len(input) {
		t.Errorf("Write n = %d, expected %d", n, len(input))
	}

	if err != nil {
		t.Errorf("got err '%v', expected nil", err)
	}

LINES1:
	for {
		select {
		case line := <-lines:
			gotLines = append(gotLines, line)
		default:
			break LINES1
		}
	}

	if diffs := deep.Equal(gotLines, expectLines); diffs != nil {
		t.Error(diffs)
	}
}

func TestStreamingLineBuffering(t *testing.T) {
	// Lines not terminated with newline are held in the line buffer until next
	// write. When line is later terminated with newline, we prepend the buffered
	// line and send the complete line.
	lines := make(chan string, 1)
	out := cmd.NewOutputStream(lines)

	// Write 3 unterminated lines. Without a newline, they'll be buffered until...
	for i := 0; i < 3; i++ {
		input := "foo"
		n, err := out.Write([]byte(input))

		if err != nil {
			t.Errorf("got err '%v', expected nil", err)
		}

		if n != len(input) {
			t.Errorf("Write n = %d, expected %d", n, len(input))
		}

		// Should not get a line yet because it's not newline terminated
		var gotLine string
		select {
		case gotLine = <-lines:
			t.Errorf("got line '%s', expected no line yet", gotLine)
		default:
		}
	}

	// Write a line with newline that terminate the previous input
	input := "bar\n"
	n, err := out.Write([]byte(input))

	if err != nil {
		t.Errorf("got err '%v', expected nil", err)
	}

	if n != len(input) {
		t.Errorf("Write n = %d, expected %d", n, len(input))
	}

	// Now we get the previously buffered part of the line "foofoofoo" plus
	// the newline terminated part "bar"
	var gotLine string
	select {
	case gotLine = <-lines:
	default:
		t.Fatal("blocked receiving line")
	}

	expectLine := "foofoofoobar"
	if gotLine != expectLine {
		t.Errorf("got line '%s', expected '%s'", gotLine, expectLine)
	}
}

func TestStreamingErrLineBufferOverflow1(t *testing.T) { // nolint funlen
	// Overflow the line buffer in 1 write. The first line "bc" is sent,
	// but the remaining line can't be buffered because it's +2 bytes larger
	// than the line buffer.
	longLine := make([]byte, 3+cmd.DefaultLineBufferSize+2) // "bc\nAAA...zz"
	longLine[0] = 'b'
	longLine[1] = 'c'
	longLine[2] = '\n'

	for i := 3; i < cmd.DefaultLineBufferSize; i++ {
		longLine[i] = 'A'
	}

	// These 2 chars cause ErrLineBufferOverflow:
	longLine[cmd.DefaultLineBufferSize] = 'z'
	longLine[cmd.DefaultLineBufferSize+1] = 'z'

	lines := make(chan string, 5)
	out := cmd.NewOutputStream(lines)

	// Write the long line, it should only write (n) 3 bytes for "bc\n"
	n, err := out.Write(longLine)

	if n != 3 { // "bc\n"
		t.Errorf("Write n = %d, expected 3", n)
	}

	switch errt := err.(type) {
	case cmd.ErrLineBufferOverflow:
		if errt.BufferSize != cmd.DefaultLineBufferSize {
			t.Errorf("ErrLineBufferOverflow.BufferSize = %d, expected %d", errt.BufferSize, cmd.DefaultLineBufferSize)
		}

		if errt.BufferFree != cmd.DefaultLineBufferSize {
			t.Errorf("ErrLineBufferOverflow.BufferFree = %d, expected %d", errt.BufferFree, cmd.DefaultLineBufferSize)
		}

		if errt.Line != string(longLine[3:]) {
			t.Errorf("ErrLineBufferOverflow.Line = '%s', expected '%s'", errt.Line, string(longLine[3:]))
		}

		if errt.Error() == "" {
			t.Errorf("ErrLineBufferOverflow.Error() string is empty, expected something")
		}
	default:
		t.Errorf("got err '%v', expected cmd.ErrLineBufferOverflow", err)
	}

	// "bc" should be sent before the overflow error
	var gotLine string
	select {
	case gotLine = <-lines:
	default:
		t.Fatal("blocked on <-lines")
	}

	if gotLine != "bc" {
		t.Errorf("got line '%s', expected 'bc'", gotLine)
	}

	// Streaming should still work after an overflow. However, Go is going to
	// stop any time Write() returns an error.
	n, err = out.Write([]byte("foo\n"))

	if n != 4 {
		t.Errorf("got n %d, expected 4", n)
	}

	if err != nil {
		t.Errorf("got err '%v', expected nil", err)
	}

	select {
	case gotLine = <-lines:
	default:
		t.Fatal("blocked on <-lines")
	}

	if gotLine != "foo" {
		t.Errorf("got line: '%s', expected 'foo'", gotLine)
	}
}

func TestStreamingErrLineBufferOverflow2(t *testing.T) {
	// Overflow line buffer on 2nd write. So first write puts something in the
	// buffer, and then 2nd overflows it instead of completing the line.
	lines := make(chan string, 1)
	out := cmd.NewOutputStream(lines)

	// Get "bar" into the buffer by omitting its newline
	input := "foo\nbar"
	n, err := out.Write([]byte(input))

	if err != nil {
		t.Errorf("got err '%v', expected nil", err)
	}

	if n != len(input) {
		t.Errorf("Write n = %d, expected %d", n, len(input))
	}

	// Only "foo" sent, not "bar" yet
	var gotLine string
	select {
	case gotLine = <-lines:
	default:
		t.Fatal("blocked on <-lines")
	}

	if gotLine != "foo" {
		t.Errorf("got line '%s', expected 'foo'", gotLine)
	}

	// Buffer contains "bar", now wverflow it on 2nd write
	longLine := make([]byte, cmd.DefaultLineBufferSize)
	for i := 0; i < cmd.DefaultLineBufferSize; i++ {
		longLine[i] = 'X'
	}

	n, err = out.Write(longLine)
	if n != 0 {
		t.Errorf("Write n = %d, expected 0", n)
	}

	switch errt := err.(type) {
	case cmd.ErrLineBufferOverflow:
		// Buffer has "bar" so it's free is total - 3
		if errt.BufferFree != cmd.DefaultLineBufferSize-3 {
			t.Errorf("ErrLineBufferOverflow.BufferFree = %d, expected %d", errt.BufferFree, cmd.DefaultLineBufferSize)
		}
		// Up to but not include "bc\n" because it should have been truncated
		expectLine := "bar" + string(longLine)
		if errt.Line != expectLine {
			t.Errorf("ErrLineBufferOverflow.Line = '%s', expected '%s'", errt.Line, expectLine)
		}
	default:
		t.Errorf("got err '%v', expected cmd.ErrLineBufferOverflow", err)
	}
}

func TestStreamingSetLineBufferSize(t *testing.T) {
	// Same overflow as TestStreamingErrLineBufferOverflow1 but before we use
	// stream output, we'll increase buffer size by calling SetLineBufferSize
	// which should prevent the overflow
	longLine := make([]byte, 3+cmd.DefaultLineBufferSize+2) // "bc\nAAA...z\n"
	longLine[0] = 'b'
	longLine[1] = 'c'
	longLine[2] = '\n'

	for i := 3; i < cmd.DefaultLineBufferSize; i++ {
		longLine[i] = 'A'
	}

	longLine[cmd.DefaultLineBufferSize] = 'z'
	longLine[cmd.DefaultLineBufferSize+1] = '\n'

	lines := make(chan string, 5)
	out := cmd.NewOutputStream(lines)
	out.SetLineBufferSize(cmd.DefaultLineBufferSize * 2)

	n, err := out.Write(longLine)

	if err != nil {
		t.Errorf("error '%v', expected nil", err)
	}

	if n != len(longLine) {
		t.Errorf("Write n = %d, expected %d", n, len(longLine))
	}

	// First we get "bc"
	var gotLine string
	select {
	case gotLine = <-lines:
	default:
		t.Fatal("blocked on <-lines")
	}

	if gotLine != "bc" {
		t.Errorf("got line '%s', expected 'bc'", gotLine)
	}

	// Then we get the long line because the buffer was large enough to hold it
	select {
	case gotLine = <-lines:
	default:
		t.Fatal("blocked on <-lines")
	}

	expectLine := string(longLine[3 : cmd.DefaultLineBufferSize+1]) // not newline

	if gotLine != expectLine {
		t.Errorf("got line: '%s', expected '%s'", gotLine, expectLine)
	}
}
