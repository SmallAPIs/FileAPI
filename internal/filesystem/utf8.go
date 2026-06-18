package filesystem

import (
	"io"
	"os"
	"sync"
	"unicode/utf8"
)

const readChunkSize = 64 * 1024

var readBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, readChunkSize)
		return &b
	},
}

// readValidatedUTF8 reads up to maxSize bytes from path and validates UTF-8 incrementally.
func readValidatedUTF8(path string, maxSize int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bufPtr := readBufPool.Get().(*[]byte)
	defer readBufPool.Put(bufPtr)
	chunk := *bufPtr

	var out []byte
	var pending []byte

	for {
		n, readErr := f.Read(chunk)
		if n > 0 {
			combined := append(pending, chunk[:n]...)
			if int64(len(out)+len(combined)) > maxSize {
				return nil, ErrTooLarge
			}

			valid, remain, err := consumeValidUTF8(combined)
			if err != nil {
				return nil, err
			}
			out = append(out, valid...)
			pending = remain
		}

		if readErr == io.EOF {
			if len(pending) > 0 {
				return nil, ErrBinaryContent
			}
			return out, nil
		}
		if readErr != nil {
			return nil, readErr
		}
	}
}

// streamValidatedUTF8 copies UTF-8 text from path to w, failing on invalid sequences.
func streamValidatedUTF8(path string, maxSize int64, w io.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	bufPtr := readBufPool.Get().(*[]byte)
	defer readBufPool.Put(bufPtr)
	chunk := *bufPtr

	var pending []byte
	var written int64

	for {
		n, readErr := f.Read(chunk)
		if n > 0 {
			combined := append(pending, chunk[:n]...)
			if written+int64(len(combined)) > maxSize {
				return ErrTooLarge
			}

			valid, remain, err := consumeValidUTF8(combined)
			if err != nil {
				return err
			}
			if len(valid) > 0 {
				if _, err := w.Write(valid); err != nil {
					return err
				}
				written += int64(len(valid))
			}
			pending = remain
		}

		if readErr == io.EOF {
			if len(pending) > 0 {
				return ErrBinaryContent
			}
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

// consumeValidUTF8 returns validated prefix bytes and any incomplete trailing sequence.
func consumeValidUTF8(data []byte) (valid, remain []byte, err error) {
	i := 0
	for i < len(data) {
		if !utf8.RuneStart(data[i]) {
			return nil, nil, ErrBinaryContent
		}
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size == 1 {
			return nil, nil, ErrBinaryContent
		}
		if i+size > len(data) {
			return data[:i], data[i:], nil
		}
		i += size
	}
	return data, nil, nil
}
