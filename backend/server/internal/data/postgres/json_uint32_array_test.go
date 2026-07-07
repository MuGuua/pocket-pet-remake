package postgres

import "testing"

func TestUnmarshalFlexibleUint32Array(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    []uint32
		wantErr bool
	}{
		{
			name:  "numeric array",
			input: `[101,102,103]`,
			want:  []uint32{101, 102, 103},
		},
		{
			name:  "string array",
			input: `["101","102","103"]`,
			want:  []uint32{101, 102, 103},
		},
		{
			name:  "mixed array",
			input: `[101,"102",103]`,
			want:  []uint32{101, 102, 103},
		},
		{
			name:    "invalid string item",
			input:   `["abc"]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := unmarshalFlexibleUint32Array([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("unexpected length: got=%d want=%d", len(got), len(tt.want))
			}
			for index := range tt.want {
				if got[index] != tt.want[index] {
					t.Fatalf("unexpected value at index %d: got=%d want=%d", index, got[index], tt.want[index])
				}
			}
		})
	}
}
