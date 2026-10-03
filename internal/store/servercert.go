package store

import (
	"context"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-go"
)

// サーバー証明書の出所。
const (
	CertSourceSelfSigned = "self-signed"
	CertSourceUploaded   = "uploaded"
)

// CertSlot はサーバー証明書の保存先。
type CertSlot string

const (
	// CertSlotAV は認証ベクターAPI 用のサーバー証明書。
	CertSlotAV CertSlot = "cfg:av_tls"
	// CertSlotAdmin は管理API 用のサーバー証明書。
	CertSlotAdmin CertSlot = "cfg:admin_tls"
)

// ServerCertificate はサーバー証明書と秘密鍵。
type ServerCertificate struct {
	CertPEM   []byte
	KeyPEM    []byte
	Source    string
	UpdatedAt time.Time
}

// KEYS: 保存先
// ARGV: cert_pem, key_pem, source, updated_at
var initServerCertificateScript = valkey.NewLuaScript(`
if redis.call('EXISTS', KEYS[1]) == 1 then
  return 0
end
redis.call('HSET', KEYS[1], 'cert_pem', ARGV[1], 'key_pem', ARGV[2], 'source', ARGV[3], 'updated_at', ARGV[4])
return 1
`)

// ServerCertificate は保存されているサーバー証明書を返す。なければ ErrNotFound を返す。
func (s *Store) ServerCertificate(ctx context.Context, slot CertSlot) (ServerCertificate, error) {
	f, err := s.c.Do(ctx, s.c.B().Hgetall().Key(string(slot)).Build()).AsStrMap()
	if err != nil {
		return ServerCertificate{}, fmt.Errorf("get server certificate: %w", err)
	}
	if len(f) == 0 {
		return ServerCertificate{}, ErrNotFound
	}
	return ServerCertificate{
		CertPEM:   []byte(f["cert_pem"]),
		KeyPEM:    []byte(f["key_pem"]),
		Source:    f["source"],
		UpdatedAt: parseTime(f["updated_at"]),
	}, nil
}

// InitServerCertificate は、サーバー証明書がまだ保存されていない場合に限り保存する。
// 既にあれば何もせず false を返す。
func (s *Store) InitServerCertificate(ctx context.Context, slot CertSlot, certPEM, keyPEM []byte, source string) (bool, error) {
	args := []string{string(certPEM), string(keyPEM), source, now()}
	n, err := initServerCertificateScript.Exec(ctx, s.c, []string{string(slot)}, args).AsInt64()
	if err != nil {
		return false, fmt.Errorf("init server certificate: %w", err)
	}
	return n == 1, nil
}

// SetServerCertificate はサーバー証明書を置き換える。
func (s *Store) SetServerCertificate(ctx context.Context, slot CertSlot, certPEM, keyPEM []byte, source string) error {
	cmd := s.c.B().Hset().Key(string(slot)).FieldValue().
		FieldValue("cert_pem", string(certPEM)).
		FieldValue("key_pem", string(keyPEM)).
		FieldValue("source", source).
		FieldValue("updated_at", now()).Build()
	if err := s.c.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("set server certificate: %w", err)
	}
	return nil
}
