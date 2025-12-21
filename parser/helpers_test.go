package parser_test

import (
	"testing"

	"github.com/4nd3r5on/go-envfile/parser"
)

func TestExtractKey(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		equalIdx int
		want     parser.KeyData
		wantErr  bool
	}{
		// Basic key extraction
		{
			name:     "simple key",
			line:     "KEY=value",
			equalIdx: 3,
			want: parser.KeyData{
				Key:   "KEY",
				Start: 0,
				End:   2,
			},
			wantErr: false,
		},
		{
			name:     "lowercase key",
			line:     "key=value",
			equalIdx: 3,
			want: parser.KeyData{
				Key:   "key",
				Start: 0,
				End:   2,
			},
			wantErr: false,
		},
		{
			name:     "mixed case key",
			line:     "MyKey=value",
			equalIdx: 5,
			want: parser.KeyData{
				Key:   "MyKey",
				Start: 0,
				End:   4,
			},
			wantErr: false,
		},
		{
			name:     "key with underscore",
			line:     "MY_KEY=value",
			equalIdx: 6,
			want: parser.KeyData{
				Key:   "MY_KEY",
				Start: 0,
				End:   5,
			},
			wantErr: false,
		},
		{
			name:     "key with numbers",
			line:     "KEY123=value",
			equalIdx: 6,
			want: parser.KeyData{
				Key:   "KEY123",
				Start: 0,
				End:   5,
			},
			wantErr: false,
		},
		{
			name:     "key with mixed alphanumeric and underscore",
			line:     "my_key_123=value",
			equalIdx: 10,
			want: parser.KeyData{
				Key:   "my_key_123",
				Start: 0,
				End:   9,
			},
			wantErr: false,
		},

		// Keys with trailing spaces before equals
		{
			name:     "key with one trailing space",
			line:     "KEY =value",
			equalIdx: 4,
			want: parser.KeyData{
				Key:   "KEY",
				Start: 0,
				End:   2,
			},
			wantErr: false,
		},
		{
			name:     "key with multiple trailing spaces",
			line:     "KEY   =value",
			equalIdx: 6,
			want: parser.KeyData{
				Key:   "KEY",
				Start: 0,
				End:   2,
			},
			wantErr: false,
		},
		{
			name:     "key with tab before equals",
			line:     "KEY\t=value",
			equalIdx: 4,
			want: parser.KeyData{
				Key:   "KEY",
				Start: 0,
				End:   2,
			},
			wantErr: false,
		},

		// Keys with leading spaces
		{
			name:     "key with leading space",
			line:     " KEY=value",
			equalIdx: 4,
			want: parser.KeyData{
				Key:   "KEY",
				Start: 1,
				End:   3,
			},
			wantErr: false,
		},
		{
			name:     "key with multiple leading spaces",
			line:     "   KEY=value",
			equalIdx: 6,
			want: parser.KeyData{
				Key:   "KEY",
				Start: 3,
				End:   5,
			},
			wantErr: false,
		},
		{
			name:     "key with leading tab",
			line:     "\tKEY=value",
			equalIdx: 4,
			want: parser.KeyData{
				Key:   "KEY",
				Start: 1,
				End:   3,
			},
			wantErr: false,
		},

		// Keys with both leading and trailing spaces
		{
			name:     "key with surrounding spaces",
			line:     "  KEY  =value",
			equalIdx: 7,
			want: parser.KeyData{
				Key:   "KEY",
				Start: 2,
				End:   4,
			},
			wantErr: false,
		},
		{
			name:     "key with mixed whitespace",
			line:     " \tKEY \t=value",
			equalIdx: 7,
			want: parser.KeyData{
				Key:   "KEY",
				Start: 2,
				End:   4,
			},
			wantErr: false,
		},

		// Single character keys
		{
			name:     "single character key",
			line:     "A=value",
			equalIdx: 1,
			want: parser.KeyData{
				Key:   "A",
				Start: 0,
				End:   0,
			},
			wantErr: false,
		},
		{
			name:     "single character key with spaces",
			line:     "  A  =value",
			equalIdx: 5,
			want: parser.KeyData{
				Key:   "A",
				Start: 2,
				End:   2,
			},
			wantErr: false,
		},

		// Long keys
		{
			name:     "very long key",
			line:     "THIS_IS_A_VERY_LONG_ENVIRONMENT_VARIABLE_KEY_NAME=value",
			equalIdx: 49,
			want: parser.KeyData{
				Key:   "THIS_IS_A_VERY_LONG_ENVIRONMENT_VARIABLE_KEY_NAME",
				Start: 0,
				End:   48,
			},
			wantErr: false,
		},

		// Error cases
		{
			name:     "equals at start of line",
			line:     "=value",
			equalIdx: 0,
			want:     parser.KeyData{},
			wantErr:  true,
		},
		{
			name:     "only spaces before equals",
			line:     "   =value",
			equalIdx: 3,
			want:     parser.KeyData{},
			wantErr:  true,
		},
		{
			name:     "only tab before equals",
			line:     "\t=value",
			equalIdx: 1,
			want:     parser.KeyData{},
			wantErr:  true,
		},
		{
			name:     "only mixed whitespace before equals",
			line:     " \t \t=value",
			equalIdx: 4,
			want:     parser.KeyData{},
			wantErr:  true,
		},

		// Keys with special characters (if supported)
		{
			name:     "key with hyphen",
			line:     "MY-KEY=value",
			equalIdx: 6,
			want: parser.KeyData{
				Key:   "MY-KEY",
				Start: 0,
				End:   5,
			},
			wantErr: false,
		},
		{
			name:     "key with dot",
			line:     "MY.KEY=value",
			equalIdx: 6,
			want: parser.KeyData{
				Key:   "MY.KEY",
				Start: 0,
				End:   5,
			},
			wantErr: false,
		},

		// Keys in longer lines with values
		{
			name:     "key with quoted value",
			line:     "KEY=\"some value\"",
			equalIdx: 3,
			want: parser.KeyData{
				Key:   "KEY",
				Start: 0,
				End:   2,
			},
			wantErr: false,
		},
		{
			name:     "key with empty value",
			line:     "KEY=",
			equalIdx: 3,
			want: parser.KeyData{
				Key:   "KEY",
				Start: 0,
				End:   2,
			},
			wantErr: false,
		},
		{
			name:     "key with complex value",
			line:     "DATABASE_URL=postgresql://user:pass@localhost:5432/db",
			equalIdx: 12,
			want: parser.KeyData{
				Key:   "DATABASE_URL",
				Start: 0,
				End:   11,
			},
			wantErr: false,
		},

		// Edge cases with positioning
		{
			name:     "key at different position in line",
			line:     "export KEY=value",
			equalIdx: 10,
			want: parser.KeyData{
				Key:   "KEY",
				Start: 7,
				End:   9,
			},
			wantErr: false,
		},
		{
			name:     "key after multiple tokens",
			line:     "some prefix KEY=value",
			equalIdx: 15,
			want: parser.KeyData{
				Key:   "KEY",
				Start: 12,
				End:   14,
			},
			wantErr: false,
		},

		// Keys with numbers at different positions
		{
			name:     "key starting with number (if allowed)",
			line:     "123KEY=value",
			equalIdx: 6,
			want: parser.KeyData{
				Key:   "123KEY",
				Start: 0,
				End:   5,
			},
			wantErr: false,
		},
		{
			name:     "key with only numbers",
			line:     "12345=value",
			equalIdx: 5,
			want: parser.KeyData{
				Key:   "12345",
				Start: 0,
				End:   4,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := parser.ExtractKey(tt.line, tt.equalIdx)

			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ExtractKey() failed: %v", gotErr)
				}
				return
			}

			if tt.wantErr {
				t.Fatal("ExtractKey() succeeded unexpectedly")
			}

			// Compare all fields
			if got.Key != tt.want.Key {
				t.Errorf("ExtractKey().Key = %q, want %q", got.Key, tt.want.Key)
			}
			if got.Start != tt.want.Start {
				t.Errorf("ExtractKey().Start = %d, want %d", got.Start, tt.want.Start)
			}
			if got.End != tt.want.End {
				t.Errorf("ExtractKey().End = %d, want %d", got.End, tt.want.End)
			}
		})
	}
}
