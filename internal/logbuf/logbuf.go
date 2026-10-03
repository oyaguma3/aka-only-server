// Package logbuf は直近のログをメモリ上のリングバッファに保持する。
// 管理API の /logs がここから読む。
package logbuf

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"
)

// Entry はバッファに保持するログ 1 件。
type Entry struct {
	Seq   int64 // 起動からの通し番号。1 始まり
	Time  time.Time
	Level string
	Msg   string
	Attrs map[string]any
}

// Buffer は直近のログを保持するリングバッファ。
type Buffer struct {
	mu      sync.Mutex
	entries []Entry // 容量に達するまでは追記し、達したら next の位置を上書きする
	next    int
	seq     int64
}

// New は capacity 件を保持するバッファを作る。
func New(capacity int) *Buffer {
	return &Buffer{entries: make([]Entry, 0, max(capacity, 1))}
}

func (b *Buffer) add(e Entry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seq++
	e.Seq = b.seq
	if len(b.entries) < cap(b.entries) {
		b.entries = append(b.entries, e)
		return
	}
	b.entries[b.next] = e
	b.next = (b.next + 1) % len(b.entries)
}

// snapshot は保持しているログを古い順に返す。
func (b *Buffer) snapshot() []Entry {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Entry, 0, len(b.entries))
	out = append(out, b.entries[b.next:]...)
	return append(out, b.entries[:b.next]...)
}

// After は通し番号が after より大きいログを、古い順に最大 limit 件返す。
func (b *Buffer) After(after int64, limit int) []Entry {
	all := b.snapshot()
	i := 0
	for i < len(all) && all[i].Seq <= after {
		i++
	}
	return all[i:min(i+limit, len(all))]
}

// Latest は最新の limit 件を古い順に返す。
func (b *Buffer) Latest(limit int) []Entry {
	all := b.snapshot()
	return all[max(len(all)-limit, 0):]
}

// Handler は、ログを inner に渡しつつ Buffer にも記録する slog.Handler を返す。
func (b *Buffer) Handler(inner slog.Handler) slog.Handler {
	return &handler{buf: b, inner: inner}
}

type handler struct {
	buf    *Buffer
	inner  slog.Handler
	attrs  []slog.Attr // WithAttrs で付けられた属性。キーはグループ名を含めて展開済み
	prefix string      // WithGroup で開いたグループ名（末尾に . を含む）
}

func (h *handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *handler) Handle(ctx context.Context, r slog.Record) error {
	e := Entry{Time: r.Time.UTC(), Level: r.Level.String(), Msg: r.Message}
	if len(h.attrs) > 0 || r.NumAttrs() > 0 {
		e.Attrs = make(map[string]any, len(h.attrs)+r.NumAttrs())
		for _, a := range h.attrs {
			e.Attrs[a.Key] = a.Value.Any()
		}
		r.Attrs(func(a slog.Attr) bool {
			flatten(h.prefix, a, func(a slog.Attr) { e.Attrs[a.Key] = a.Value.Any() })
			return true
		})
	}
	h.buf.add(e)
	return h.inner.Handle(ctx, r)
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := *h
	h2.inner = h.inner.WithAttrs(attrs)
	h2.attrs = slices.Clone(h.attrs)
	for _, a := range attrs {
		flatten(h.prefix, a, func(a slog.Attr) { h2.attrs = append(h2.attrs, a) })
	}
	return &h2
}

func (h *handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := *h
	h2.inner = h.inner.WithGroup(name)
	h2.prefix = h.prefix + name + "."
	return &h2
}

// flatten は属性を「グループ名.キー」の形に展開し、値を JSON にできる形へそろえて emit に渡す。
func flatten(prefix string, a slog.Attr, emit func(slog.Attr)) {
	v := a.Value.Resolve()
	if v.Kind() == slog.KindGroup {
		p := prefix
		if a.Key != "" {
			p += a.Key + "."
		}
		for _, ga := range v.Group() {
			flatten(p, ga, emit)
		}
		return
	}
	if a.Key == "" {
		return
	}
	var out any
	switch v.Kind() {
	case slog.KindString, slog.KindInt64, slog.KindUint64, slog.KindFloat64, slog.KindBool:
		out = v.Any()
	case slog.KindTime:
		out = v.Time().UTC().Format(time.RFC3339Nano)
	case slog.KindDuration:
		out = v.Duration().String()
	default:
		switch x := v.Any().(type) {
		case error:
			out = x.Error()
		case []string:
			out = x
		default:
			out = v.String()
		}
	}
	emit(slog.Any(prefix+a.Key, out))
}
