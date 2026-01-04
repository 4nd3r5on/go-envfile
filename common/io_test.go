package common_test

import (
	"bufio"
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/4nd3r5on/go-envfile/common"
)

func TestReadLine(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    []byte
		wantLen int64
		wantErr bool
	}{
		{
			name:    "simple line with LF",
			data:    "hello\n",
			want:    []byte("hello"),
			wantLen: 6,
			wantErr: false,
		},
		{
			name:    "simple line with CRLF",
			data:    "hello\r\n",
			want:    []byte("hello"),
			wantLen: 7,
			wantErr: false,
		},
		{
			name:    "line without newline",
			data:    "hello",
			want:    []byte("hello"),
			wantLen: 5,
			wantErr: false,
		},
		{
			name:    "empty line with LF",
			data:    "\n",
			want:    []byte(""),
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "empty line with CRLF",
			data:    "\r\n",
			want:    []byte(""),
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "empty string",
			data:    "",
			want:    []byte(""),
			wantLen: 0,
			wantErr: true, // EOF
		},
		{
			name:    "line with spaces",
			data:    "  hello world  \n",
			want:    []byte("  hello world  "),
			wantLen: 16,
			wantErr: false,
		},
		{
			name:    "line with special characters",
			data:    "hello\tworld\x00test\n",
			want:    []byte("hello\tworld\x00test"),
			wantLen: 17,
			wantErr: false,
		},
		{
			name:    "unicode content",
			data:    "hello 世界\n",
			want:    []byte("hello 世界"),
			wantLen: 13, // "hello " (6) + "世界" (6) + "\n" (1)
			wantErr: false,
		},
		{
			name:    "very long line",
			data:    strings.Repeat("a", 10000) + "\n",
			want:    []byte(strings.Repeat("a", 10000)),
			wantLen: 10001,
			wantErr: false,
		},
		{
			name:    "multiple lines - only first read",
			data:    "first\nsecond\nthird\n",
			want:    []byte("first"),
			wantLen: 6,
			wantErr: false,
		},
		{
			name:    "line with only CR (not CRLF)",
			data:    "hello\rworld\n",
			want:    []byte("hello\rworld"),
			wantLen: 12,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got2, gotErr := common.ReadLine(
				bufio.NewReader(strings.NewReader(tt.data)),
			)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ReadLine() failed: %v", gotErr)
				}

				return
			}

			if tt.wantErr {
				t.Fatal("ReadLine() succeeded unexpectedly")
			}

			if !bytes.Equal(got, tt.want) {
				t.Errorf("ReadLine() content = %q, want %q", got, tt.want)
			}

			if got2 != tt.wantLen {
				t.Errorf("ReadLine() length = %v, want %v", got2, tt.wantLen)
			}
		})
	}
}

func TestCopySpan(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	tests := []struct {
		name      string
		data      string
		w         *bytes.Buffer
		start     int64
		end       int64
		curOffset *int64
		wantErr   bool
	}{
		{
			name:      "zero length span",
			data:      "hello world",
			w:         &bytes.Buffer{},
			start:     5,
			end:       5,
			curOffset: ptr(int64(0)),
			wantErr:   false,
		},
		{
			name:      "full content copy",
			data:      "hello",
			w:         &bytes.Buffer{},
			start:     0,
			end:       5,
			curOffset: ptr(int64(0)),
			wantErr:   false,
		},
		{
			name:      "middle span",
			data:      "hello world",
			w:         &bytes.Buffer{},
			start:     6,
			end:       11,
			curOffset: ptr(int64(0)),
			wantErr:   false,
		},
		{
			name:      "already at correct offset",
			data:      "hello world",
			w:         &bytes.Buffer{},
			start:     6,
			end:       11,
			curOffset: ptr(int64(6)),
			wantErr:   false,
		},
		{
			name:      "span beyond file length",
			data:      "hello",
			w:         &bytes.Buffer{},
			start:     0,
			end:       100,
			curOffset: ptr(int64(0)),
			wantErr:   true,
		},
		{
			name:      "invalid seek position",
			data:      "hello",
			w:         &bytes.Buffer{},
			start:     -1,
			end:       5,
			curOffset: ptr(int64(0)),
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := common.CopySpan(strings.NewReader(tt.data), tt.w, tt.start, tt.end, tt.curOffset, logger)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("CopySpan() failed: %v", gotErr)
				}

				return
			}

			if tt.wantErr {
				t.Fatal("CopySpan() succeeded unexpectedly")
			}

			// Verify content was copied correctly (for non-error cases)
			if tt.end > tt.start {
				expected := tt.end - tt.start
				if int64(tt.w.Len()) != expected {
					t.Errorf("expected %d bytes written, got %d", expected, tt.w.Len())
				}
			}
		})
	}
}

// Helper function to create int64 pointers.
func ptr(i int64) *int64 {
	return &i
}
