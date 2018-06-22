package bomstrip

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"testing"
)

func TestBomstrip(t *testing.T) {
	testReader := func(input io.Reader, expect []byte, expectError error) {
		buf, err := ioutil.ReadAll(input)
		if err != expectError {
			t.Fatalf("expected error %v, saw %v, expecting bytes %x\n", expectError, err, expect)
		}
		if bytes.Compare(buf, expect) != 0 {
			t.Fatalf("expected bytes %x, saw %x\n", expect, buf)
		}
	}
	testBytes := func(input []byte, expect []byte, expectError error) {
		t.Helper()
		testReader(NewReader(bytes.NewReader(input)), expect, expectError)
	}

	type Test struct {
		Input  []byte
		Expect []byte
	}

	bigbuf := make([]byte, 3+128*1024)
	for i := range bigbuf {
		bigbuf[i] = 0x20
	}
	copy(bigbuf, bom)

	var tests = []Test{
		// non-BOMs
		{[]byte{}, []byte{}},
		{[]byte{0x20}, []byte{0x20}},
		{[]byte{0x20, 0x21}, []byte{0x20, 0x21}},
		{[]byte{0x20, 0x21, 0x22}, []byte{0x20, 0x21, 0x22}},
		{[]byte{0x20, 0x21, 0x22, 0x23}, []byte{0x20, 0x21, 0x22, 0x23}},

		// partial BOM prefix, should be passed on unchanged
		{bom[:1], bom[:1]},
		{bom[:2], bom[:2]},
		{[]byte{0xEF, 0x20}, []byte{0xEF, 0x20}},
		{[]byte{0xEF, 0xBB, 0x20}, []byte{0xEF, 0xBB, 0x20}},

		// just a BOM, should be stripped
		{bom[:3], []byte{}},

		// BOM followed by regular data, should be stripped
		{[]byte{0xEF, 0xBB, 0xBF, 0x20}, []byte{0x20}},
		{bigbuf, bigbuf[3:]},
	}

	for _, t := range tests {
		testBytes(t.Input, t.Expect, nil)
	}

	for _, t := range tests {
		testReader(NewReader(&singleByteReader{t.Input}), t.Expect, nil)
	}

	xerr := errors.New("test")
	for _, t := range tests {
		testReader(NewReader(&errorReader{t.Input, xerr}), t.Expect, xerr)
	}
}

type singleByteReader struct {
	buf []byte
}

func (r *singleByteReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(r.buf) == 0 {
		return 0, io.EOF
	}
	p[0] = r.buf[0]
	r.buf = r.buf[1:]
	return 1, nil
}

type errorReader struct {
	buf []byte
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	if len(r.buf) > 0 {
		n = copy(p, r.buf)
		r.buf = r.buf[n:]
		return
	}
	return 0, r.err
}
