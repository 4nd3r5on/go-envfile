package common

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"slices"
)

// ByteSpan = [start,end] byte offsets
type ByteSpan struct {
	Start int64
	End   int64
}

// Pass 1: scan file, compute byte offsets for every line.
// Only store offsets for lines that appear in `patches`.
func ScanLineOffsets(path string, patches map[int64]Patch) (map[int64]ByteSpan, error) {
	out := make(map[int64]ByteSpan, len(patches))

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)

	var (
		offset  int64 = 0
		lineIdx int64 = 0
	)

	for {
		_, fullLen, err := ReadWholeLine(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if _, exists := patches[lineIdx]; exists {
			out[lineIdx] = ByteSpan{
				Start: offset,
				End:   offset + fullLen,
			}
		}

		offset += fullLen
		lineIdx++
	}

	return out, nil
}

// ReadWholeLine: returns complete line (no trailing newline) and
// the total byte length *including* the newline.
// Works even for huge lines split by Reader.ReadLine.
//
// Length accounting is critical for correct offsets.
func ReadWholeLine(r *bufio.Reader) ([]byte, int64, error) {
	var (
		buf    []byte
		total  int64
		prefix bool
	)

	line, pfx, err := r.ReadLine()
	if err != nil {
		return nil, 0, err
	}
	prefix = pfx

	buf = append(buf, line...)
	total += int64(len(line))

	for prefix {
		line, pfx, err = r.ReadLine()
		if err != nil {
			return nil, 0, err
		}
		prefix = pfx
		buf = append(buf, line...)
		total += int64(len(line))
	}

	// newline
	total += 1
	return buf, total, nil
}

// CopySpan: copies file[start:end] verbatim into w.
func CopySpan(file *os.File, w io.Writer, start, end int64, curOffset *int64) error {
	if *curOffset != start {
		if _, err := file.Seek(start, io.SeekStart); err != nil {
			return err
		}
		*curOffset = start
	}

	toCopy := end - start
	_, err := io.CopyN(w, file, toCopy)
	if err != nil {
		return err
	}

	*curOffset = end
	return nil
}

// WriteLine: writes a full text line + newline.
func WriteLine(w io.Writer, s string) error {
	_, err := io.WriteString(w, s+"\n")
	return err
}

// ApplySinglePatch: writes insertion + either original line or skip.
func ApplySinglePatch(file *os.File, w io.Writer, p Patch, span ByteSpan, curOffset *int64) error {
	// insert first
	if p.Insert != "" {
		if err := WriteLine(w, p.Insert); err != nil {
			return err
		}
	}

	if p.RemoveLine {
		// skip original
		if _, err := file.Seek(span.End, io.SeekStart); err != nil {
			return err
		}
		*curOffset = span.End
		return nil
	}

	// keep original
	return CopySpan(file, w, span.Start, span.End, curOffset)
}

// Orchestrator: reads spans, streams file, applies patches, writes temp, renames.
func ApplyPatches(path string, patches map[int64]Patch) error {
	if len(patches) == 0 {
		return nil
	}

	// Pass 1
	spans, err := ScanLineOffsets(path, patches)
	if err != nil {
		return err
	}

	// Sort line numbers
	lines := make([]int64, 0, len(patches))
	for ln := range patches {
		lines = append(lines, ln)
	}
	slices.Sort(lines)

	// Open original
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()

	// Create temp
	tmp, err := os.CreateTemp(filepath.Dir(path), ".patch-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	w := bufio.NewWriter(tmp)

	// copy pointer
	var curOffset int64 = 0

	// Process patches
	for _, ln := range lines {
		span := spans[ln]
		p := patches[ln]

		// copy everything before this line
		if err := CopySpan(in, w, curOffset, span.Start, &curOffset); err != nil {
			return err
		}

		// apply the patch
		if err := ApplySinglePatch(in, w, p, span, &curOffset); err != nil {
			return err
		}
	}

	// copy the tail (after last patch)
	info, _ := in.Stat()
	if curOffset < info.Size() {
		if err := CopySpan(in, w, curOffset, info.Size(), &curOffset); err != nil {
			return err
		}
	}

	// flush/close
	if err := w.Flush(); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}

	// preserve permissions
	if st, err := os.Stat(path); err == nil {
		tmp.Chmod(st.Mode())
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	// atomic replace
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}

	return nil
}
