package engine

import "testing"

func TestFirstNonEmptyLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "single", in: "health check failed", want: "health check failed"},
		{name: "multiline", in: "container x is restarting\nstack line", want: "container x is restarting"},
		{name: "leading blanks", in: "\n  \nshort\nlong", want: "short"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := firstNonEmptyLine(tt.in)
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestSplitLogMessage(t *testing.T) {
	t.Parallel()

	got := splitLogMessage("a\n\n b \n\nc")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %q want %q", i, got[i], want[i])
		}
	}

	if empty := splitLogMessage("\n  \n"); len(empty) != 0 {
		t.Fatalf("expected empty slice, got %#v", empty)
	}
}
