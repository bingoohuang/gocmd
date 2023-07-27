package linestream

import (
	"bytes"
	"fmt"
)

// LineStream represents real time, line by line output from a running Cmd.
// Lines are terminated by a single newline preceded by an optional carriage
// return. Both newline and carriage return are stripped from the line when
// sent to a caller-provided channel.
//
// The caller must begin receiving before starting the Cmd. Write blocks on the
// channel; the caller must always read the channel. The channel is not closed
// by the LineStream.
//
// A Cmd in this package uses an LineStream for both STDOUT and STDERR when
// created by calling NewCmdOptions and Options.Streaming is true. To use
// LineStream directly with a Go standard library os/exec.Command:
//
//	import "os/exec"
//	import "github.com/gobars/cmd"
//
//	stdoutChan := make(chan string, 100)
//	go func() {
//	    for line := range stdoutChan {
//	        // Do something with the line
//	    }
//	}()
//
//	runnableCmd := exec.Command(...)
//	stdout := cmd.New(stdoutChan)
//	runnableCmd.Stdout = stdout
//
// While runnableCmd is running, lines are sent to the channel as soon as they
// are written and newline-terminated by the command. After the command finishes,
// the caller should wait for the last lines to be sent:
//
//	for len(stdoutChan) > 0 {
//	    time.Sleep(10 * time.Millisecond)
//	}
//
// Since the channel is not closed by the LineStream, the two indications that
// all lines have been sent and received are the command finishing and the
// channel size being zero.
type LineStream struct {
	lineProcessor LineProcessor
	buf           []byte
	bufSize       int
	lastChar      int
}

type LineProcessor func(line string)

// New creates a new streaming output on the given channel. The
// caller must begin receiving on the channel before the command is started.
// The LineStream never closes the channel.
func New(lineProcessor LineProcessor) *LineStream {
	out := &LineStream{
		lineProcessor: lineProcessor,
		bufSize:       DefaultLineBufferSize,
		buf:           make([]byte, DefaultLineBufferSize),
		lastChar:      0,
	}

	return out
}

// Write makes LineStream implement the io.Writer interface. Do not call
// this function directly.
func (rw *LineStream) Write(p []byte) (n int, err error) {
	n = len(p) // end of buffer
	firstCharPos := 0

LINES:
	for {
		// Find next newline in stream buffer. nextLine starts at 0, but buff
		// can contain multiple lines, like "foo\nbar". So in that case nextLine
		// will be 0 ("foo\nbar\n") then 4 ("bar\n") on next iteration. And i
		// will be 3 and 7, respectively. So lines are [0:3] are [4:7].
		newlineOffset := bytes.IndexByte(p[firstCharPos:], '\n')
		if newlineOffset < 0 {
			break LINES // no newline in stream, next line incomplete
		}

		// End of line offset is start (nextLine) + newline offset. Like bufio.Scanner,
		// we allow \r\n but strip the \r too by decrementing the offset for that byte.
		lastChar := firstCharPos + newlineOffset // "line\n"
		if newlineOffset > 0 && p[newlineOffset-1] == '\r' {
			lastChar-- // "line\r\n"
		}

		// Send the line, prepend line buffer if set
		var line string
		if rw.lastChar > 0 {
			line = string(rw.buf[0:rw.lastChar])
			rw.lastChar = 0 // reset buffer
		}
		line += string(p[firstCharPos:lastChar])
		rw.lineProcessor(line) // blocks if chan full

		// Next line offset is the first byte (+1) after the newline (i)
		firstCharPos += newlineOffset + 1
	}

	if firstCharPos < n {
		remain := len(p[firstCharPos:])
		bufFree := len(rw.buf[rw.lastChar:])

		if remain > bufFree {
			var line string
			if rw.lastChar > 0 {
				line = string(rw.buf[0:rw.lastChar])
			}

			line += string(p[firstCharPos:])
			err = ErrLineBufferOverflow{
				Line:       line,
				BufferSize: rw.bufSize,
				BufferFree: bufFree,
			}
			n = firstCharPos

			return // implicit
		}

		copy(rw.buf[rw.lastChar:], p[firstCharPos:])
		rw.lastChar += remain
	}

	return n, err // implicit
}

// SetLineBufferSize sets the internal line buffer size. The default is DEFAULT_LINE_BUFFER_SIZE.
// This function must be called immediately after New, and it is not
// safe to call by multiple goroutines.
//
// Increasing the line buffer size can help reduce ErrLineBufferOverflow errors.
func (rw *LineStream) SetLineBufferSize(n int) { rw.bufSize = n; rw.buf = make([]byte, rw.bufSize) }

// --------------------------------------------------------------------------

const (
	// DefaultLineBufferSize is the default size of the LineStream line buffer.
	// The default value is usually sufficient, but if ErrLineBufferOverflow errors
	// occur, try increasing the size by calling OutputBuffer.SetLineBufferSize.
	DefaultLineBufferSize = 16384
)

// ErrLineBufferOverflow is returned by LineStream.Write when the internal
// line buffer is filled before a newline character is written to terminate a
// line. Increasing the line buffer size by calling LineStream.SetLineBufferSize
// can help prevent this error.
type ErrLineBufferOverflow struct {
	Line       string // Unterminated line that caused the error
	BufferSize int    // Internal line buffer size
	BufferFree int    // Free bytes in line buffer
}

func (e ErrLineBufferOverflow) Error() string {
	return fmt.Sprintf("line does not contain newline and is %d bytes too long to buffer (buffer size: %d)",
		len(e.Line)-e.BufferSize, e.BufferSize)
}
