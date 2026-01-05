//revive:disable:var-naming
package common

//revive:enable:var-naming

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
)

func patchAddNewLine(p Patch) Patch {
	p.Insert = p.Insert + "\n"
	p.InsertAfter = p.InsertAfter + "\n"

	return p
}

// ScanLineOffsets is a wrapper for ScanLineOffsetsReader.
func ScanLineOffsets(path string, patches map[int64]Patch, logger *slog.Logger) (map[int64]ByteSpan, error) {
	logger.Info("scanning line offsets", "path", path, "patch_count", len(patches))

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return ScanLineOffsetsReader(bufio.NewReader(f), patches, logger)
}

// Pass 1: scan file, compute byte offsets for every line.
// Only store offsets for lines that appear in `patches`.
func ScanLineOffsetsReader(r Reader, patches map[int64]Patch, logger *slog.Logger) (map[int64]ByteSpan, error) {
	if logger == nil {
		logger = slog.Default()
	}

	out := make(map[int64]ByteSpan, len(patches))

	var (
		offset  int64
		lineIdx int64
	)

	for {
		_, fullLen, err := ReadLine(r)
		if errors.Is(err, io.EOF) {
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

			logger.Debug("found patch line", "line", lineIdx, "start", offset, "end", offset+fullLen, "length", fullLen)
		}

		offset += fullLen
		lineIdx++
	}

	// Allow a patch that targets the EOF insertion point (index == lineIdx).
	if _, exists := patches[lineIdx]; exists {
		out[lineIdx] = ByteSpan{
			Start: offset,
			End:   offset, // zero-length span at EOF
		}

		logger.Debug("found patch at EOF", "line", lineIdx, "start", offset, "end", offset)
	}

	// Validate there are no patches beyond EOF (index > lineIdx).
	for idx := range patches {
		if idx > lineIdx {
			return nil, fmt.Errorf("patch index %d beyond EOF (%d lines)", idx, lineIdx)
		}
	}

	logger.Info("scan complete", "total_lines", lineIdx, "spans_found", len(out))

	return out, nil
}

// ApplySinglePatch: writes insertion + either original line or skip.
func ApplySinglePatch(file io.ReadSeeker, w io.Writer, p Patch, span ByteSpan, curOffset *int64, logger *slog.Logger) error {
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

func ProcessPatches(
	in io.ReadSeeker,
	size int64,
	w io.Writer,
	spans map[int64]ByteSpan,
	patches map[int64]Patch,
	logger *slog.Logger,
) error {
	if logger == nil {
		logger = slog.Default()
	}

	lines := make([]int64, 0, len(patches))
	for ln := range patches {
		lines = append(lines, ln)
	}

	slices.Sort(lines)

	logger.Info("processing patches", "count", len(lines), "lines", lines)

	var curOffset int64

	for _, ln := range lines {
		span, ok := spans[ln]
		if !ok {
			return fmt.Errorf("missing span for line %d", ln)
		}

		p := patches[ln]

		logger.Debug(
			"processing patch",
			"line", ln,
			"span_start", span.Start,
			"span_end", span.End,
			"current_offset", curOffset,
		)

		if span.Start < curOffset {
			return fmt.Errorf("overlapping patch at line %d", ln)
		}

		logger.Debug(
			"copying pre-span",
			"from", curOffset,
			"to", span.Start,
		)

		err := CopySpan(in, w, curOffset, span.Start, &curOffset, logger)
		if err != nil {
			return err
		}

		logger.Debug("applying patch", "line", ln)

		err = ApplySinglePatch(in, w, p, span, &curOffset, logger)
		if err != nil {
			return err
		}
	}

	if curOffset < size {
		logger.Debug(
			"copying tail",
			"from", curOffset,
			"to", size,
		)

		err := CopySpan(in, w, curOffset, size, &curOffset, logger)
		if err != nil {
			return err
		}
	} else {
		logger.Debug(
			"no tail to copy",
			"current_offset", curOffset,
			"file_size", size,
		)
	}

	logger.Info("patch processing complete", "final_offset", curOffset)

	return nil
}

// Orchestrator: reads spans, streams file, applies patches, writes temp, renames.
func ApplyPatches(path string, patches map[int64]Patch, autoNewLine bool, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	logger.Info("starting patch application",
		"path", path,
		"patch_count", len(patches),
		"auto_newline", autoNewLine)

	if len(patches) == 0 {
		logger.Debug("no patches to apply")

		return nil
	}

	if autoNewLine {
		logger.Debug("applying auto-newline to patches")

		for i, p := range patches {
			patches[i] = patchAddNewLine(p)
		}
	}

	logger.Debug("scanning line offsets")

	spans, err := ScanLineOffsets(path, patches, logger)
	if err != nil {
		logger.Error("failed to scan line offsets", "error", err)

		return err
	}

	logger.Debug("line offsets scanned successfully", "span_count", len(spans))

	logger.Debug("opening input file", "path", path)

	in, err := os.Open(path)
	if err != nil {
		logger.Error("failed to open input file", "path", path, "error", err)

		return err
	}

	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		logger.Error("failed to stat input file", "path", path, "error", err)

		return err
	}

	logger.Debug("input file info", "size", info.Size(), "mode", info.Mode())

	tmpDir := filepath.Dir(path)
	logger.Debug("creating temporary file", "dir", tmpDir)

	tmp, err := os.CreateTemp(tmpDir, ".patch-*.tmp")
	if err != nil {
		logger.Error("failed to create temporary file", "dir", tmpDir, "error", err)

		return err
	}

	tmpName := tmp.Name()
	logger.Debug("temporary file created", "tmp_path", tmpName)

	buf := bufio.NewWriter(tmp)

	logger.Info("processing patches")

	if err := ProcessPatches(
		in,
		info.Size(),
		buf,
		spans,
		patches,
		logger,
	); err != nil {
		logger.Error("failed to process patches", "error", err)

		return err
	}

	logger.Debug("patches processed successfully")

	logger.Debug("flushing buffer")

	if err := buf.Flush(); err != nil {
		logger.Error("failed to flush buffer", "error", err)

		return err
	}

	logger.Debug("syncing temporary file")

	if err := tmp.Sync(); err != nil {
		logger.Error("failed to sync temporary file", "error", err)

		return err
	}

	if st, err := os.Stat(path); err == nil {
		logger.Debug("preserving file permissions", "mode", st.Mode())
		_ = tmp.Chmod(st.Mode())
	}

	logger.Debug("closing temporary file")

	if err := tmp.Close(); err != nil {
		logger.Error("failed to close temporary file", "error", err)

		return err
	}

	logger.Debug("renaming temporary file to target", "from", tmpName, "to", path)

	if err := os.Rename(tmpName, path); err != nil {
		logger.Error("failed to rename temporary file", "from", tmpName, "to", path, "error", err)

		return err
	}

	logger.Info("patch application completed successfully", "path", path)

	return nil
}
