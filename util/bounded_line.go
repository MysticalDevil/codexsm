package util

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

// ReadBoundedLine reads one logical line without allowing unbounded growth.
// If the line exceeds maxBytes, the remainder is discarded through the next
// newline and truncated is reported as true.
func ReadBoundedLine(r *bufio.Reader, maxBytes int) (line []byte, truncated bool, err error) {
	if maxBytes <= 0 {
		maxBytes = 1
	}

	var out bytes.Buffer
	for {
		chunk, readErr := r.ReadSlice('\n')
		if len(chunk) > 0 {
			remaining := maxBytes - out.Len()
			if remaining > 0 {
				if len(chunk) > remaining {
					_, _ = out.Write(chunk[:remaining])
					truncated = true
				} else {
					_, _ = out.Write(chunk)
				}
			} else {
				truncated = true
			}
		}

		switch {
		case readErr == nil:
			line = bytes.TrimSpace(out.Bytes())
			return line, truncated, nil
		case errors.Is(readErr, bufio.ErrBufferFull):
			continue
		case errors.Is(readErr, io.EOF):
			line = bytes.TrimSpace(out.Bytes())
			return line, truncated, io.EOF
		default:
			return nil, truncated, readErr
		}
	}
}
