package formatter

import (
	"io"
	"sync"
)

// Gate holds back writes until it is opened. Curl writes headers to stderr and
// the body to stdout, which are read concurrently, so the body is gated until
// the headers have been written to keep them in order.
type Gate struct {
	Out io.Writer

	mu   sync.Mutex
	open bool
	buf  []byte
}

func (g *Gate) Write(p []byte) (n int, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.open {
		g.buf = append(g.buf, p...)
		return len(p), nil
	}
	return g.Out.Write(p)
}

// Open flushes the held back writes and lets the next ones through.
func (g *Gate) Open() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.open = true
	if len(g.buf) == 0 {
		return nil
	}
	_, err := g.Out.Write(g.buf)
	g.buf = nil
	return err
}

// Close holds back the next writes until Open is called.
func (g *Gate) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.open = false
}
