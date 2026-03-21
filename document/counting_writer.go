package document

import "io"

// countingWriter wraps an io.Writer and tracks the total number of bytes written.
type countingWriter struct {
	w       io.Writer
	written int64
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.written += int64(n)
	return n, err
}
