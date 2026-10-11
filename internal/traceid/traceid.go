// Package traceid は、要求のトレースID（X-Trace-ID ヘッダー）を扱う。
// 認証ベクターAPI と管理API で共通に使う。
package traceid

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
)

// Header はトレースID を運ぶヘッダー。
const Header = "X-Trace-ID"

// pattern は受け付けるトレースID（印字可能 ASCII 1〜64 文字）。それ以外はサーバーが採番し直す。
// 本PoC（eapaka-radius-server-poc）の provisioning-api と vector-gateway と同じ。
var pattern = regexp.MustCompile(`^[\x21-\x7E]{1,64}$`)

// New はトレースID（16 バイトの乱数の 16 進表記）を採番する。
func New() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Resolve は、ヘッダーの値が受け付けられる形ならそのまま返し、そうでなければ（空を含む）採番して返す。
func Resolve(v string) string {
	if pattern.MatchString(v) {
		return v
	}
	return New()
}
