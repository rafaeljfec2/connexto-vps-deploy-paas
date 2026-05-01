package docker

import (
	"reflect"
	"testing"
	"time"
)

func TestBuildLogsArgs(t *testing.T) {
	since := time.Date(2026, 4, 30, 19, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		container string
		tail      int
		since     *time.Time
		follow    bool
		want      []string
	}{
		{
			name:      "static tail without since uses default 100",
			container: "myapp",
			tail:      0,
			since:     nil,
			follow:    false,
			want:      []string{"logs", "--tail", "100", "--timestamps", "myapp"},
		},
		{
			name:      "static tail honours custom value",
			container: "myapp",
			tail:      250,
			since:     nil,
			follow:    false,
			want:      []string{"logs", "--tail", "250", "--timestamps", "myapp"},
		},
		{
			name:      "static tail with since appends RFC3339 flag",
			container: "myapp",
			tail:      50,
			since:     &since,
			follow:    false,
			want:      []string{"logs", "--tail", "50", "--timestamps", "--since", "2026-04-30T19:00:00Z", "myapp"},
		},
		{
			name:      "follow mode without since stays backwards-compat",
			container: "myapp",
			tail:      0,
			since:     nil,
			follow:    true,
			want:      []string{"logs", "-f", "--tail", "100", "--timestamps", "myapp"},
		},
		{
			name:      "follow mode with since appends RFC3339 flag",
			container: "myapp",
			tail:      0,
			since:     &since,
			follow:    true,
			want:      []string{"logs", "-f", "--tail", "100", "--timestamps", "--since", "2026-04-30T19:00:00Z", "myapp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildLogsArgs(tt.container, tt.tail, tt.since, tt.follow)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildLogsArgs() mismatch\nwant: %v\n got: %v", tt.want, got)
			}
		})
	}
}

func TestBuildLogsArgs_NormalisesSinceToUTC(t *testing.T) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}
	since := time.Date(2026, 4, 30, 16, 0, 0, 0, loc)

	got := buildLogsArgs("myapp", 100, &since, false)

	const wantSinceArg = "2026-04-30T19:00:00Z"
	for i, arg := range got {
		if arg == "--since" {
			if i+1 >= len(got) {
				t.Fatalf("--since missing value: %v", got)
			}
			if got[i+1] != wantSinceArg {
				t.Fatalf("--since value not normalised to UTC RFC3339\nwant: %s\n got: %s", wantSinceArg, got[i+1])
			}
			return
		}
	}
	t.Fatalf("--since not present in args: %v", got)
}
