// Package bomstrip helps removing UTF-8 byte order marks, BOM.
package bomstrip

import (
	"io"
)

type reader struct {
	bomBytesSeen int // -1 when there is no bom
	passthrough  []byte
	reader       io.Reader
}

// NewReader wraps a reader that skips a leading utf-8 byte order mark (BOM).
// Only full BOMs will be stripped. Partial BOM's will pass. Errors will pass.
// Data is only read from the underlying reader as requested.
// Once the leading BOM has been skipped or found to be absent, reads are simply passed on to the underlying reader.
func NewReader(r io.Reader) io.Reader {
	return &reader{reader: r}
}

var bom = []byte{0xEF, 0xBB, 0xBF}

// Read tries hard not to cause unexpected behaviour. It doesn't read more or differently than the caller asked for.
// If the underlying reader returns an error, it is passed through, including the partially read buffer.
func (r *reader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	for r.bomBytesSeen >= 0 && r.bomBytesSeen < len(bom) {
		xn := len(bom) - r.bomBytesSeen
		if xn > len(p) {
			xn = len(p)
		}
		buf := make([]byte, xn)
		n, err = r.reader.Read(buf)
		if err != nil && err != io.EOF {
			// copy the data that we've read so far
			nn := copy(p, bom[:r.bomBytesSeen])
			p = p[nn:]
			if n > 0 {
				nn += copy(p, buf[:n])
			}
			return nn, err
		}
		if n == 0 {
			r.passthrough = bom[:r.bomBytesSeen]
			r.bomBytesSeen = -1
			break
		}
		for n > 0 && r.bomBytesSeen != len(bom) {
			if buf[0] != bom[r.bomBytesSeen] {
				// pass through the parts of the BOM we have seen, and the remaining data
				r.passthrough = make([]byte, r.bomBytesSeen+n)
				copy(r.passthrough, bom[:r.bomBytesSeen])
				copy(r.passthrough[r.bomBytesSeen:], buf[:n])
				r.bomBytesSeen = -1
				break
			}
			r.bomBytesSeen++
			buf = buf[1:]
			n--
		}
	}
	np := len(r.passthrough)
	if np > 0 {
		np := copy(p, r.passthrough[:np])
		r.passthrough = r.passthrough[np:]
		return np, nil
	}
	return r.reader.Read(p)
}
