package core

import (
	"fmt"
	"io"
)

// MaxReadSize is the default maximum size for bounded reads (100 MB).
const MaxReadSize = 100 * 1024 * 1024

// LimitedReadAll reads from r up to maxSize bytes. If maxSize is <= 0,
// MaxReadSize is used. Returns an error if the data exceeds the limit.
func LimitedReadAll(r io.Reader, maxSize int64) ([]byte, error) {
	if maxSize <= 0 {
		maxSize = MaxReadSize
	}
	lr := io.LimitReader(r, maxSize+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxSize {
		return nil, fmt.Errorf("data exceeds maximum size of %d bytes", maxSize)
	}
	return data, nil
}
