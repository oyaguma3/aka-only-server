package traceid

import (
	"regexp"
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	generated := regexp.MustCompile(`^[0-9a-f]{32}$`)
	for _, tc := range []struct {
		name, in string
		keep     bool
	}{
		{"そのまま使う", "abc-123_XYZ.~", true},
		{"64 文字まで", strings.Repeat("a", 64), true},
		{"空なら採番", "", false},
		{"65 文字は採番", strings.Repeat("a", 65), false},
		{"空白を含むと採番", "a b", false},
		{"制御文字を含むと採番", "a\nb", false},
		{"ASCII 以外は採番", "トレース", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Resolve(tc.in)
			if tc.keep {
				if got != tc.in {
					t.Errorf("Resolve(%q) = %q, want unchanged", tc.in, got)
				}
				return
			}
			if !generated.MatchString(got) {
				t.Errorf("Resolve(%q) = %q, want generated id", tc.in, got)
			}
		})
	}
	if New() == New() {
		t.Error("New returned the same id twice")
	}
}
