# はじめに
このAPIサーバーは、下記の設計ドキュメントを Google Antigravity (Gemini 3 pro - High) にプロンプトとして投入して作られたものです。Google Antigravityのトライアルとして作らせてみただけのもので、ほとんど動作確認をしておらず、実際に利用できるかは保証しません。  
なお、/docs 配下にユーザーズガイドとAPI仕様ドキュメントを出力させているので、本当に使ってみたい方はこれらを参照してみてください。  

-----
# アプリ設計ドキュメント

### ■アプリケーションのコンセプト

指定したIMSIに対して、MilenageアルゴリズムによるAKA認証ベクターを返す「APIサーバー」を開発する。
このAPIサーバーは、EAP-AKA対応RADIUS/EAPサーバーとして動作する別のアプリに対して、指定されたIMSIの認証ベクターを払い出す役割を持つ。

### ■使用言語・ライブラリなど

- 開発言語として Go を使用する。
    - 環境変数ファイルを取り扱うため、パッケージ joho/godotenv を利用すること。
    - Milenageアルゴリズム処理のため、パッケージ wmnsk/milenage を利用すること。
    - Webフレームワークとして Gin を利用すること。
- ローカルのデータベースとして PostgreSQL を使用する。
    - Go言語のpostgreSQL用パッケージとして jackc/pgx を利用すること。

### ■要件

##### （アプリ本体）

- アプリ本体が単一の実行バイナリーとなるようコンパイルできること。
- 基本動作は「APIリクエストでIMSIを（リソースURIで）指定し、レスポンスで認証ベクターを返す」とする。
- 実行時に systemd にサービスとして登録され、systemctl start/stop/restart などのコマンドで操作可能とする。
    - これは、アプリをバックグラウンドで稼働させることを主要な目的とする。

##### （加入者データベース）

- IMSIと認証鍵情報(ki/opc/sqn/amf)は、データベースに格納すること。
    - 各要素は平文での格納を許容する。
- データベースにはPostgreSQLを使い、以下のSQLコマンドで作成されたデータベース＆テーブル＆ユーザーが作成されている前提とする。
    - 参考情報
        - データベース名：akaserverdb
        - テーブル名：subscribers (スキーマはpublic)
        - ユーザー名：akaserver
    - SQLコマンドは以下に記載。

```sql
-- =========================================================
-- 1. データベースの作成
-- =========================================================
CREATE DATABASE akaserverdb;

-- =========================================================
-- 2. データベースへの接続切り替え
-- =========================================================
-- 作成したデータベースの中にテーブルや権限を作っていきます。
\c akaserverdb

-- =========================================================
-- 3. テーブルの作成 (subscribers)
-- =========================================================
CREATE TABLE public.subscribers (
    -- IMSI: 15桁の数字 (Primary Key)
    imsi VARCHAR(15) PRIMARY KEY,

    -- Ki: 16byte = 32文字 (Hex string)
    ki   VARCHAR(32) NOT NULL,

    -- OPC: 16byte = 32文字 (Hex string)
    opc  VARCHAR(32) NOT NULL,

    -- SQN: 6byte = 12文字 (Hex string)
    sqn  VARCHAR(12) NOT NULL,

    -- AMF: 2byte = 4文字 (Hex string)
    amf  VARCHAR(4)  NOT NULL,

    -- (管理用) レコード作成日時
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Goアプリのエラーを防ぐためのフォーマット制約
    -- 10進数のみ、または16進数(0-9, a-f, A-F)のみ許可
    CONSTRAINT chk_imsi_format CHECK (imsi ~ '^[0-9]{15}$'),
    CONSTRAINT chk_ki_hex      CHECK (ki  ~ '^[0-9a-fA-F]{32}$'),
    CONSTRAINT chk_opc_hex     CHECK (opc ~ '^[0-9a-fA-F]{32}$'),
    CONSTRAINT chk_sqn_hex     CHECK (sqn ~ '^[0-9a-fA-F]{12}$'),
    CONSTRAINT chk_amf_hex     CHECK (amf ~ '^[0-9a-fA-F]{4}$')
);

-- =========================================================
-- 4. アプリ用ユーザーの作成
-- =========================================================
CREATE USER akaserver WITH PASSWORD 'akaserver';

-- =========================================================
-- 5. 権限の付与
-- =========================================================

-- (1) データベースへの接続許可
GRANT CONNECT ON DATABASE akaserverdb TO akaserver;

-- (2) publicスキーマの使用許可
GRANT USAGE ON SCHEMA public TO akaserver;

-- (3) テーブル操作許可 (読み取り・作成・更新・削除のみ)
-- DDL (CREATE/DROP TABLE) は許可されません
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO akaserver;

-- (4) 将来作成されるテーブルへの自動権限設定
ALTER DEFAULT PRIVILEGES IN SCHEMA public 
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO akaserver;
```

##### （認証ベクター）

- 認証ベクターは加入者ごとに計算し、加入者はIMSIで識別する。
- 認証ベクターの計算に使うパラメーターは以下であるとする。3GPP TS 33.102 を参照すること。
    - ki
        - 16byte長
    - opc
        - 16byte長
    - sqn
        - 6byte長
        - SQN(48bit) = SEQ(43bit) || IND(5bit)
        - SQN管理および処理は、3GPP TS 33.102 - C.3.2 Profile 2: management of sequence numbers which are not time-based に準拠すること。
    - amf
        - 2byte長
        - 原則として 0x8000 が格納される想定。
- 認証ベクターの計算に使うパラメーターは、加入者データベースに格納する。

##### （API）

- REST APIを可能な限り満たすAPI仕様であること。
- API利用のための認証処理は実装しないが、以下の要素を環境変数ファイルで指定可能とすること。
    - API利用を許可するIPアドレス＆ポート番号
        - IPアドレス＆ポート番号のセットは、複数登録できるようにすること。
        - 認証ベクター払い出し用APIエンドポイントと加入者データベース操作用APIエンドポイントで別々に指定できるようにすること。
- 認証ベクター払い出し用APIエンドポイントは、通常処理と再同期処理で統一すること。
    - IMSIを用いてリソースURIを分離すること。
    - 通常処理と再同期処理の判別は、リクエストボディにAUTSとRANDが含まれているかどうかで行うこと。
- 加入者データベースへの操作について
    - IMSIと認証用鍵情報(ki/opc/sqn/amf)は、REST APIで閲覧・登録・変更・削除が可能であること。
    - データベースやスキーマ、テーブルそのものに対する操作は禁止する。
    - 別途開発するWeb UIアプリケーション（Go/htmx/Alpine.js/Tailwind CSSでの開発を想定）からAPI経由で操作することも想定しておくこと。

##### （AKA処理）

- 基本的に、3GPP標準仕様 TS 33.102 に準拠する。
- AKA処理における認証ベクターとは、以下の5つを指す。
    - AUTN
    - RAND
    - XRES
    - CK
    - IK
- AKAにおける再同期処理に対応すること。
- AKA処理にはMilenageアルゴリズムを用いること。
- AKA処理におけるCおよびRは、3GPP TS 35.206の標準値を使用すること。
- 認証ベクター生成後にSQNを更新する際は、SEQのみインクリメントしてINDは維持すること。

##### （ログファイル）

- 標準ライブラリーの log/slog を利用し、出力ログを構造化すること。
- 出力ログのローテーション機能を具備すること。
    - 利用するGoパッケージとして natefinch/lumberjack を推奨する。
- 環境変数ファイルでログ出力先やローテーションの各種設定を指定できること。
    - デフォルトのログ出力先は、実行バイナリーと同じ場所とする。

### ■その他

- アプリ利用のためのガイドラインを、Markdown形式のドキュメントとして作成すること。
    - APIをcurlで実行する際のサンプルコマンドを記載すること。
- このアプリで利用可能なAPIに対する、API仕様ドキュメントをmarkdown形式で作成すること。
    - 環境変数ファイルで複数のclient IP addressが設定可能であり、その設定サンプルを記載すること。
    - 各APIに対するcurlコマンドの実行サンプルと、そのレスポンス例を記載すること。
        - ただし、各APIの仕様に追記するのではなく、別の章を立てて記載すること。
