package common

import (
	"bytes"
	"io"
	"log/slog"
)

type Reader interface {
	io.Reader
	ReadBytes(delim byte) ([]byte, error)
	ReadByte() (byte, error)
	ReadLine() (line []byte, isPrefix bool, err error)
	Peek(n int) ([]byte, error)
}

func ReadLineWithEOL(reader Reader) ([]byte, error) {
	var buf bytes.Buffer

	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF && buf.Len() > 0 {
				// Return last line without newline
				return buf.Bytes(), nil
			}
			return nil, err
		}

		buf.WriteByte(b)

		// If LF appears, we end the line (handles LF and CRLF)
		if b == '\n' {
			return buf.Bytes(), nil
		}

		if b == '\r' {
			next, err := reader.Peek(1)
			if err == nil && len(next) == 1 && next[0] == '\n' {
				reader.ReadByte()   // consume \n
				buf.WriteByte('\n') // append it
			}
			return buf.Bytes(), nil
		}
	}
}

// ReadLine: returns complete line (no trailing newline) and
// the total byte length *including* the newline if present.
// Uses Reader.ReadByte and Reader.Peek to detect CRLF.
func ReadLine(r Reader) ([]byte, int64, error) {
	var buf []byte

	for {
		b, err := r.ReadByte()
		if err != nil {
			if err == io.EOF {
				if len(buf) == 0 {
					// nothing to return
					return nil, 0, io.EOF
				}
				// EOF after some bytes — no trailing newline
				return buf, int64(len(buf)), nil
			}
			return nil, 0, err
		}

		// Line feed => newline (LF)
		if b == '\n' {
			return buf, int64(len(buf)) + 1, nil
		}

		// Carriage return: could be CRLF or literal CR
		if b == '\r' {
			// Peek to see if next byte is '\n'
			p, perr := r.Peek(1)
			if perr == nil && len(p) > 0 && p[0] == '\n' {
				// consume the '\n' and treat CRLF as newline of length 2
				_, rerr := r.ReadByte()
				if rerr != nil {
					return nil, 0, rerr
				}
				return buf, int64(len(buf)) + 2, nil
			}
			// Not a CRLF sequence — treat '\r' as regular byte
			buf = append(buf, b)
			continue
		}

		// Regular byte -> append to buffer
		buf = append(buf, b)
	}
}

// CopySpan: copies file[start:end] verbatim into w.
func CopySpan(file io.ReadSeeker, w io.Writer, start, end int64, curOffset *int64, logger *slog.Logger) error {
	length := end - start
	logger.Debug("copy span", "start", start, "end", end, "length", length, "current_offset", *curOffset)

	if length == 0 {
		logger.Debug("skip copy - zero length")
		return nil
	}

	if *curOffset != start {
		logger.Debug("seeking to start", "from", *curOffset, "to", start)
		if _, err := file.Seek(start, io.SeekStart); err != nil {
			return err
		}
		*curOffset = start
	}

	toCopy := end - start
	n, err := io.CopyN(w, file, toCopy)
	if err != nil {
		return err
	}

	logger.Debug("copied bytes", "count", n)
	*curOffset = end
	return nil
}
