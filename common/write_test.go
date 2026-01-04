package common_test

import (
	"bufio"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/4nd3r5on/go-envfile/common"
)

func TestScanLineOffsetsReader(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	tests := []struct {
		name    string
		input   string
		patches map[int64]common.Patch
		want    map[int64]common.ByteSpan
		wantErr bool
	}{
		{
			name:    "no patches",
			input:   "a\nb\nc\n",
			patches: map[int64]common.Patch{},
			want:    map[int64]common.ByteSpan{},
		},
		{
			name:  "single patch",
			input: "a\nb\nc\n",
			patches: map[int64]common.Patch{
				1: {},
			},
			want: map[int64]common.ByteSpan{
				1: {Start: 2, End: 4}, // "b\n"
			},
		},
		{
			name:  "multiple patches",
			input: "aa\nbbb\nc\n",
			patches: map[int64]common.Patch{
				0: {},
				2: {},
			},
			want: map[int64]common.ByteSpan{
				0: {Start: 0, End: 3}, // "aa\n"
				2: {Start: 7, End: 9}, // "c\n"
			},
		},
		{
			name:  "last line without newline",
			input: "a\nb\nc",
			patches: map[int64]common.Patch{
				2: {},
			},
			want: map[int64]common.ByteSpan{
				2: {Start: 4, End: 5},
			},
		},
		{
			name:  "patch index out of range",
			input: "a\nb\n",
			patches: map[int64]common.Patch{
				5: {},
			},
			want: map[int64]common.ByteSpan{},
		},
		{
			name:  "patch index out of range",
			input: "a\nb\n",
			patches: map[int64]common.Patch{
				0: {},
			},
			want: map[int64]common.ByteSpan{
				0: {Start: 0, End: 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))

			got, err := common.ScanLineOffsetsReader(r, tt.patches, logger)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("unexpected error: %v", err)
				}

				return
			}

			if tt.wantErr {
				t.Fatal("expected error, got nil")
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ScanLineOffsetsReader() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
