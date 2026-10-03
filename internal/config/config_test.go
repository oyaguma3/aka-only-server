package config

import (
	"strings"
	"testing"
)

func TestParseAdminClients(t *testing.T) {
	fp := strings.Repeat("ab", 32)
	colons := strings.ToUpper(strings.Join(strings.Split(strings.Repeat("ab ", 32), " ")[:32], ":"))

	got, err := ParseAdminClients("bff=" + fp + ", ops = " + strings.Repeat("cd", 32))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[fp] != "bff" || got[strings.Repeat("cd", 32)] != "ops" {
		t.Errorf("got %v", got)
	}

	// openssl の出力形式（大文字、コロン区切り）も受け付ける。
	got, err = ParseAdminClients("bff=" + colons)
	if err != nil {
		t.Fatal(err)
	}
	if got[fp] != "bff" {
		t.Errorf("colon form: got %v", got)
	}

	if got, err := ParseAdminClients(""); err != nil || len(got) != 0 {
		t.Errorf("empty: got %v, err %v", got, err)
	}

	for _, bad := range []string{
		fp,                                // 識別名がない
		"bff=" + fp[:10],                  // 短い
		"b ff=" + fp,                      // 識別名に空白
		"a=" + fp + ",b=" + fp,            // 同じフィンガープリント
		"bff=" + strings.Repeat("zz", 32), // 16進でない
	} {
		if _, err := ParseAdminClients(bad); err == nil {
			t.Errorf("ParseAdminClients(%q): want error", bad)
		}
	}
}
