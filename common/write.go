package common

import (
	"bufio"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
)

// ByteSpan = [start,end] byte offsets
type ByteSpan struct {
	Start int64
	End   int64
}

func patchAddNewLine(p Patch) Patch {
	p.Insert = p.Insert + "\n"
	p.InsertAfter = p.InsertAfter + "\n"
	return p
}

// Pass 1: scan file, compute byte offsets for every line.
// Only store offsets for lines that appear in `patches`.
func ScanLineOffsets(path string, patches map[int64]Patch, logger *slog.Logger) (map[int64]ByteSpan, error) {
	logger.Info("scanning line offsets", "path", path, "patch_count", len(patches))
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
			span := ByteSpan{
				Start: offset,
				End:   offset + fullLen,
			}
			out[lineIdx] = span
			logger.Debug("found patch line", "line", lineIdx, "start", offset, "end", offset+fullLen, "length", fullLen)
		}

		offset += fullLen
		lineIdx++
	}

	logger.Info("scan complete", "total_lines", lineIdx, "spans_found", len(out))
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
func CopySpan(file *os.File, w io.Writer, start, end int64, curOffset *int64, logger *slog.Logger) error {
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

// ApplySinglePatch: writes insertion + either original line or skip.
func ApplySinglePatch(file *os.File, w io.Writer, p Patch, span ByteSpan, curOffset *int64, logger *slog.Logger) error {
	logger.Debug("applying patch",
		"should_insert", p.ShouldInsert,
		"remove_line", p.RemoveLine,
		"should_insert_after", p.ShouldInsertAfter,
		"insert_len", len(p.Insert),
		"insert_after_len", len(p.InsertAfter))

	// insert first
	if p.ShouldInsert {
		n, err := io.WriteString(w, p.Insert)
		if err != nil {
			return err
		}
		logger.Debug("wrote insert", "bytes", n)
	}

	if p.RemoveLine {
		// skip original
		logger.Debug("removing line, seeking to end", "from", *curOffset, "to", span.End)
		if _, err := file.Seek(span.End, io.SeekStart); err != nil {
			return err
		}
		*curOffset = span.End
	} else {
		// keep original
		logger.Debug("keeping original line")
		err := CopySpan(file, w, span.Start, span.End, curOffset, logger)
		if err != nil {
			return err
		}
	}

	// Inserting after the original
	if p.ShouldInsertAfter {
		n, err := io.WriteString(w, p.InsertAfter)
		if err != nil {
			return err
		}
		logger.Debug("wrote insert_after", "bytes", n)
	}

	return nil
}

// Orchestrator: reads spans, streams file, applies patches, writes temp, renames.
func ApplyPatches(path string, patches map[int64]Patch, autoNewLine bool, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	logger.Info("starting apply patches", "path", path, "patch_count", len(patches), "auto_newline", autoNewLine)

	if len(patches) == 0 {
		logger.Info("no patches to apply")
		return nil
	}

	if autoNewLine {
		logger.Debug("adding newlines to patches")
		for i, p := range patches {
			patches[i] = patchAddNewLine(p)
		}
	}

	// Pass 1
	spans, err := ScanLineOffsets(path, patches, logger)
	if err != nil {
		return err
	}

	// Sort line numbers
	lines := make([]int64, 0, len(patches))
	for ln := range patches {
		lines = append(lines, ln)
	}
	slices.Sort(lines)
	logger.Info("sorted patch lines", "lines", lines)

	// Open original
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}
	logger.Info("input file info", "size", info.Size(), "mode", info.Mode())

	// Create temp
	tmp, err := os.CreateTemp(filepath.Dir(path), ".patch-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	logger.Info("created temp file", "name", tmpName)

	w := bufio.NewWriter(tmp)

	// copy pointer
	var curOffset int64 = 0

	// Process patches
	for _, ln := range lines {
		span := spans[ln]
		p := patches[ln]
		logger.Info("processing patch", "line", ln, "span_start", span.Start, "span_end", span.End)

		// copy everything before this line
		if err := CopySpan(in, w, curOffset, span.Start, &curOffset, logger); err != nil {
			return err
		}

		// apply the patch
		if err := ApplySinglePatch(in, w, p, span, &curOffset, logger); err != nil {
			return err
		}
	}

	// copy the tail (after last patch)
	if curOffset < info.Size() {
		logger.Info("copying tail", "from", curOffset, "to", info.Size())
		if err := CopySpan(in, w, curOffset, info.Size(), &curOffset, logger); err != nil {
			return err
		}
	} else {
		logger.Info("no tail to copy", "current_offset", curOffset, "file_size", info.Size())
	}

	// flush/close
	logger.Debug("flushing buffer")
	if err := w.Flush(); err != nil {
		return err
	}

	logger.Debug("syncing temp file")
	if err := tmp.Sync(); err != nil {
		return err
	}

	// preserve permissions
	if st, err := os.Stat(path); err == nil {
		logger.Debug("preserving permissions", "mode", st.Mode())
		tmp.Chmod(st.Mode())
	}

	logger.Debug("closing temp file")
	if err := tmp.Close(); err != nil {
		return err
	}

	// atomic replace
	logger.Info("renaming temp to original", "from", tmpName, "to", path)
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}

	logger.Info("patches applied successfully")
	return nil
}
