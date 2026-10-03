package logbuf

import (
	"errors"
	"io"
	"log/slog"
	"slices"
	"testing"
	"time"
)

func seqs(entries []Entry) []int64 {
	out := make([]int64, len(entries))
	for i, e := range entries {
		out[i] = e.Seq
	}
	return out
}

// sink は全レベルを受け付けて出力を捨てる内側のハンドラー。
func sink() slog.Handler {
	return slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
}

func TestBufferKeepsLatest(t *testing.T) {
	b := New(3)
	log := slog.New(b.Handler(sink()))

	if got := b.Latest(10); len(got) != 0 {
		t.Errorf("empty buffer: Latest = %v", seqs(got))
	}
	for range 5 {
		log.Info("m")
	}

	// 容量を超えた分は古いものから捨てる。
	if got := seqs(b.Latest(10)); !slices.Equal(got, []int64{3, 4, 5}) {
		t.Errorf("Latest(10) = %v, want [3 4 5]", got)
	}
	if got := seqs(b.Latest(2)); !slices.Equal(got, []int64{4, 5}) {
		t.Errorf("Latest(2) = %v, want [4 5]", got)
	}
	if got := seqs(b.After(3, 10)); !slices.Equal(got, []int64{4, 5}) {
		t.Errorf("After(3, 10) = %v, want [4 5]", got)
	}
	// 捨てられた番号を指定した場合は、残っているものを先頭から返す。
	if got := seqs(b.After(0, 2)); !slices.Equal(got, []int64{3, 4}) {
		t.Errorf("After(0, 2) = %v, want [3 4]", got)
	}
	if got := seqs(b.After(5, 10)); len(got) != 0 {
		t.Errorf("After(5, 10) = %v, want empty", got)
	}
}

func TestHandlerRecordsAttrs(t *testing.T) {
	b := New(10)
	log := slog.New(b.Handler(sink())).With("listener", "av-mtls")

	log.WithGroup("req").Warn("rejected",
		"imsi", "440100123456789",
		"n", 2,
		"err", errors.New("boom"),
		"took", 1500*time.Millisecond,
		slog.Group("peer", "ip", "127.0.0.1"),
	)

	got := b.Latest(1)
	if len(got) != 1 {
		t.Fatalf("got %d entries", len(got))
	}
	e := got[0]
	if e.Level != "WARN" || e.Msg != "rejected" || e.Seq != 1 || e.Time.IsZero() {
		t.Errorf("entry = %+v", e)
	}
	want := map[string]any{
		"listener":    "av-mtls",
		"req.imsi":    "440100123456789",
		"req.n":       int64(2),
		"req.err":     "boom",
		"req.took":    "1.5s",
		"req.peer.ip": "127.0.0.1",
	}
	if len(e.Attrs) != len(want) {
		t.Errorf("Attrs = %v, want %v", e.Attrs, want)
	}
	for k, v := range want {
		if e.Attrs[k] != v {
			t.Errorf("Attrs[%q] = %#v, want %#v", k, e.Attrs[k], v)
		}
	}
}

// inner が出力しないレベルのログは、バッファにも記録しない。
func TestHandlerRespectsLevel(t *testing.T) {
	b := New(10)
	inner := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn})
	log := slog.New(b.Handler(inner))
	log.Info("dropped")
	log.Warn("kept")
	if got := b.Latest(10); len(got) != 1 || got[0].Msg != "kept" {
		t.Errorf("Latest = %+v", got)
	}
}
