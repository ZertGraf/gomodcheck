package main

import (
	"testing"
)

//goland:noinspection D
func TestParseGoListOutput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []dependency
		wantErr bool
	}{
		{
			name: "skips main module and returns updates",
			input: `{"Path":"example.com/mymod","Version":"","Main":true}
{"Path":"github.com/foo/bar","Version":"v1.0.0","Update":{"Version":"v1.2.0"}}
{"Path":"github.com/baz/qux","Version":"v0.3.0"}`,
			want: []dependency{
				{Path: "github.com/foo/bar", Current: "v1.0.0", Latest: "v1.2.0"},
			},
		},
		{
			name: "no updates available",
			input: `{"Path":"example.com/mymod","Version":"","Main":true}
{"Path":"github.com/foo/bar","Version":"v1.0.0"}`,
			want: nil,
		},
		{
			name: "multiple updates",
			input: `{"Path":"example.com/mymod","Version":"","Main":true}
{"Path":"github.com/a/b","Version":"v1.0.0","Update":{"Version":"v1.1.0"}}
{"Path":"github.com/c/d","Version":"v2.0.0","Update":{"Version":"v2.3.0"}}`,
			want: []dependency{
				{Path: "github.com/a/b", Current: "v1.0.0", Latest: "v1.1.0"},
				{Path: "github.com/c/d", Current: "v2.0.0", Latest: "v2.3.0"},
			},
		},
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:    "malformed json",
			input:   `{"Path": broken`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGoListOutput([]byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("got %d deps, want %d", len(got), len(tt.want))
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("dep[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
