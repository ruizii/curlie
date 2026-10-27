package formatter

import (
	"bytes"
	"io"
)

// HeaderCleaner removes > and < from curl --verbose output.
type HeaderCleaner struct {
	Out io.Writer

	// Verbose removes the request headers part of the output as well as the lines
	// starting with * if set to false.
	Verbose bool

	// Post is inserted after the request headers.
	Post *bytes.Buffer

	// Body, if set, is closed when a request starts and opened once the
	// response headers have been written, before curl writes the body.
	Body *Gate

	buf  []byte
	line []byte
}

var (
	capath  = []byte("  CApath:")
	ccapath = []byte("*   CApath:")
)

func (c *HeaderCleaner) Write(p []byte) (n int, err error) {
	n = len(p)
	cp := c.buf
	p = bytes.Replace(p, capath, ccapath, 1) // Fix curl misformatted line
	openBody := false
	for len(p) > 0 {
		idx := bytes.IndexByte(p, '\n')
		if idx == -1 {
			c.line = append(c.line, p...)
			break
		}
		c.line = append(c.line, p[:idx+1]...)
		p = p[idx+1:]
		ignore := false
		b, i := firstVisibleChar(c.line)
		switch b {
		case '>':
			// A new request (ie redirect): hold the body until its response.
			openBody = false
			if c.Body != nil {
				c.Body.Close()
			}
			if c.Verbose {
				c.line = c.line[i+2:]
			} else {
				ignore = true
			}
		case '<':
			c.line = c.line[i+2:]
		case '}', '{':
			ignore = true
			if b == '{' {
				// Curl logs received data before writing it to stdout.
				openBody = true
			}
			if c.Verbose && c.Post != nil {
				if post := bytes.TrimSpace(c.Post.Bytes()); len(post) > 0 {
					cp = append(append(cp, bytes.TrimSpace(c.Post.Bytes())...), '\n', '\n')
				}
				c.Post = nil
			}
		case '*':
			c.line = c.line[:0]
			if !c.Verbose {
				ignore = true
			}
		}
		if !ignore {
			cp = append(cp, c.line...)
		}
		c.line = c.line[:0]
	}
	_, err = c.Out.Write(cp)
	if openBody && c.Body != nil {
		c.Body.Open()
	}
	return
}

var colorEscape = []byte("\x1b[")

func firstVisibleChar(b []byte) (byte, int) {
	if bytes.HasPrefix(b, colorEscape) {
		if idx := bytes.IndexByte(b, 'm'); idx != -1 {
			if idx < len(b) {
				return b[idx+1], idx + 1
			} else {
				return 0, -1
			}
		}
	}
	return b[0], 0
}
