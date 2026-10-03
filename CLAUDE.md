# aka-only-server

指定した IMSI に対して、Milenage による AKA 認証ベクターを払い出す API サーバー。認証ベクターAPI（TS 29.503 GenerateAv ベース）と管理API を提供し、データは Valkey に保存する。GUI は持たない。

## 現在の状態

- フェーズ1（認証ベクターAPI）、フェーズ2（管理API）は実装済み。
- フェーズ3（BFF / Web GUI）は web-gui-for-aka-only-server で実装済み。場所は `/home/sumitakekino/projects/web-gui-for-aka-only-server`。
- 旧実装（PostgreSQL + Gin + IP 許可リスト）は削除済み。

## 最初に読むもの

| ファイル | 内容 |
|---|---|
| `docs/design-overview.md` | システム全体とこのリポジトリの設計 |
| `docs/openapi/av-api.yaml` | 認証ベクターAPI の仕様 |
| `docs/openapi/admin-api.yaml` | 管理API の仕様。BFF とサーバーの間の契約 |
| `docs/data-model.md` | Valkey のキー設計と Lua スクリプトの仕様 |
| `docs/operation-guide.md` | 導入と運用の手順 |
| `../web-gui-for-aka-only-server/CLAUDE.md` | BFF 側の方針と、検証用の実機の情報 |

## 構成

| パッケージ | 内容 |
|---|---|
| `cmd/aka-only-server` | 起動処理と、登録用のコマンド |
| `internal/aka` | Milenage によるベクター生成、AKA' の鍵導出、AUTS の検証、SQN 増加タイプ |
| `internal/store` | Valkey 上のデータ操作と Lua スクリプト |
| `internal/avapi` | 認証ベクターAPI のハンドラー |
| `internal/adminapi` | 管理API のハンドラー |
| `internal/server` | リスナーの組み立て、mTLS のクライアント照合、サーバー証明書の読み込みと差し替え |
| `internal/certs` | 証明書の生成・解析・フィンガープリント |
| `internal/logbuf` | 管理API で返すためのログのリングバッファ |
| `internal/config` | 環境変数からの設定 |

## 実装方針

- Go 1.27.1。Go のコードを書く前に `modern-go-guidelines:use-modern-go` スキルでガイドラインを確認する。
- 標準ライブラリを優先する。外部依存は次の 2 つに限る。
  - `github.com/wmnsk/milenage`: Milenage 計算
  - `github.com/valkey-io/valkey-go`: Valkey への接続
- HTTP は `net/http`（メソッドつきの `ServeMux` パターンと `r.PathValue`）。
- JSON は `encoding/json/v2`。
- ログは `log/slog` の JSON を標準出力に出す。ローテーションは Docker に任せる。
- 設定は環境変数だけで受け取る。compose の `.env` から渡す。
- コードのコメントとドキュメントは日本語で書く。

## 崩してはいけない設計上の決定（詳細は `docs/design-overview.md`）

- クライアント証明書は CA で検証せず、登録済みのフィンガープリントと照合する。
- クライアントID は再利用しない。削除後に加入者の許可集合へ古い ID が残っても安全、という前提がこれに依存している。
- クライアントの情報はメモリに持たない。ハンドシェイクとリクエストのたびに Valkey で確認し、登録・無効化・削除を即時に反映する。
- 許可判定と SQN の更新は、1 つの Lua スクリプトで原子的に行う。拒否した要求では SQN を進めない。Go 側の `SQNType.Sequence` とスクリプトは同じ計算でなければならない。
- Valkey は単一ノード前提。
- Ki、OPc、CK、IK はログにも監査ログにも出さない。Ki / OPc は管理API の専用の操作（`/subscribers/{imsi}/keys`）でだけ返す。
- 管理クライアントは環境変数で静的に設定し、全権として扱う。ユーザーや権限の概念はこのサーバーに持たない（BFF が判定する）。
- 管理API 経由の変更操作と Ki / OPc の取得は、監査ログに残す。コマンドでの操作は Valkey を直接書き換えるので、監査ログに残らない。
- 平文HTTP リスナーは既定で無効。許可は加入者単位（`allowPlain`）。

## 管理API を変えるとき

次をまとめて直す。

- `docs/openapi/admin-api.yaml`
- `internal/adminapi` の実装とテスト
- BFF 側のクライアント（web-gui-for-aka-only-server の `internal/adminapi`）と画面

BFF 側の CI は、このリポジトリの main を相手に契約テストを実行する。管理API を変えて push したら、そちらの CI も確かめる。

## 検証の進め方

- Valkey を使う結合テストは、接続先を環境変数で指定したときだけ動く。

  ```bash
  AKA_TEST_VALKEY_ADDR=127.0.0.1:16379 AKA_TEST_VALKEY_PASSWORD=<パスワード> go test ./...
  ```

  論理データベース 1 番（`internal/store`）と 2 番（`internal/adminapi`）の全データを消す。手元では専用の Valkey コンテナを別ポートで立てて使い、終わったら消す。
- 通しの確認（compose での起動、別ホストや同一ホストでの接続）は、検証用の実機で行う。接続方法と決まりは `../web-gui-for-aka-only-server/CLAUDE.md` の「検証用の実機」にある。手元の Docker では別プロジェクトのコンテナが動いているので、通しの確認には使わない。
- GitHub Actions の CI（`.github/workflows/ci.yml`）が、push のたびにテスト（Valkey の結合テストを含む）、イメージのビルド、README の手順の通し確認を実行する。
- 実装した内容は、テストが通るだけでなく、実際に起動して操作して確かめる。確かめていない点は報告に明記する。
- ドキュメントに載せる手順（コマンド）は、実際に試してから載せる。

## 進め方

- コミットは依頼されたときだけ行う。
- 設計上の判断が要る点は、番号つきの確認事項として提示し、推奨案を添える。
- 実装が設計ドキュメントとずれた場合は、ドキュメントも同じ作業の中で更新する。
