# identity-lifecycle-lab

ISO/IEC 24760-1（Identity management のフレームワーク規格）が定義するアイデンティティのライフサイクル状態管理を、実際に動くコードで学ぶための個人学習用リポジトリです。商用利用は想定していません。

## 対象とする6つの状態

| 状態 | ISO/IEC 24760-1 上の位置づけ |
|---|---|
| `enrolled` | Enrolment（登録） |
| `issued` | Registration/Issuance（発行） |
| `active` | Use/Maintenance（利用・更新） |
| `suspended` | Suspension/Reactivation（一時停止・再開） |
| `revoked` | Revocation（失効） |
| `archived` | Archiving/Deletion（保管・削除） |

許可された遷移のみを受け付け、たとえば `revoked` から `active` への直接復帰のような不正な遷移はエラーになります。状態遷移が起きるたびに「誰が・いつ・なぜ」を監査ログとして記録します。

```
enrolled → issued → active ⇄ suspended → revoked → archived
```

## アーキテクチャ

- **永続化はPostgreSQL**。`entities`（現在状態）と`transitions`（監査ログ）の2テーブル構成
- **遷移が許可されているかどうかの判定はGoアプリ側**（`internal/identity/state.go`）で行う
- **監査ログへの記録はアプリからは一切書き込まず、DBのトリガーに完全に委ねる**。`entities`へのINSERT/UPDATEに張られたトリガーが自動で`transitions`に行を追加する。アプリは同一トランザクション内で`set_config('app.actor', ...)` / `set_config('app.reason', ...)`により「誰が・なぜ」をトリガーへ引き渡すだけ。これにより、たとえ将来psqlから直接UPDATEされても監査ログが残る
- **同時遷移の競合防止**: 遷移時は対象行を`SELECT ... FOR UPDATE`でロックしてから現在状態を確認する
- **グレースフルシャットダウン**: `SIGINT`/`SIGTERM`を受けると新規リクエストの受付を止め、`http.Server.Shutdown`で処理中のリクエストを最大10秒待ってから終了し、その後にDB接続を閉じる

トリガーの定義は [docker/postgres/init.sql](docker/postgres/init.sql) を参照。

## 構成

```
internal/
  identity/
    entity.go       # Entity モデル
    transition.go   # TransitionRecord（監査ログ）モデル
    state.go        # State と許可遷移ルール
    store.go        # PostgreSQLへの読み書き（*sql.DB経由）
    id.go
  api/
    router.go       # ルーティング定義
    handlers.go      # HTTPハンドラ
docker/
  postgres/
    init.sql          # スキーマ + 監査ログ用トリガー
docker-compose.yml     # ローカルPostgreSQL
test/
  dbtest/
    dbtest.go          # テスト用DB接続ヘルパー（TEST_DATABASE_URL）
  identity/
    state_test.go      # CanTransitionのテスト（DB不要）
    store_test.go       # Storeのテスト（要PostgreSQL）
  api/
    handlers_test.go    # HTTPハンドラのテスト（要PostgreSQL）
  e2e/
    lifecycle_test.go  # 実HTTPサーバー越しの全ライフサイクル結合テスト
main.go                # HTTPサーバー起動 + グレースフルシャットダウン
```

`internal/`配下には実装コードのみを置き、テストとテスト支援コード（`dbtest`含む）はすべて`test/`配下（`test/dbtest` `test/identity` `test/api` `test/e2e`）にまとめている。これができるのは、いずれのテストも`identity_test` / `api_test`という外部テストパッケージにしており、対象パッケージの非公開要素を一切使わず公開APIのみを呼び出しているため。Goの単体テストは「対象パッケージと同じディレクトリでなければならない」という制約があるのは非公開要素にアクセスする場合のみで、公開APIしか使わない外部テストパッケージであれば`internal`の外（同一モジュール内）のどのディレクトリに置いても`go test ./...`が正しく検出・実行する。

## 動かし方

### 1. PostgreSQLを起動する

```bash
docker compose up -d
```

初回起動時に [docker/postgres/init.sql](docker/postgres/init.sql) が自動適用され、テーブルとトリガーが作成される。

### 2. サーバーを起動する

```bash
go run .
```

デフォルトでは `postgres://identity:identity@localhost:5432/identity_lifecycle?sslmode=disable` に接続する（`docker-compose.yml`の設定と一致）。接続先を変えたい場合は`DATABASE_URL`環境変数で上書きできる。

終了するときは `Ctrl+C`（`SIGINT`）または `SIGTERM` を送るとグレースフルシャットダウンする。

### エンティティを作成する

```bash
curl -s -X POST localhost:8080/entities \
  -d '{"name":"alice"}' | jq
```

### 状態遷移させる

```bash
curl -s -X POST localhost:8080/entities/<id>/transitions \
  -d '{"to":"issued","actor":"registrar","reason":"credentials issued"}' | jq
```

許可されていない遷移（例: `enrolled` から `revoked` へ直接遷移しようとする等）はHTTP 409で拒否されます。

### 遷移履歴（監査ログ）を確認する

```bash
curl -s localhost:8080/entities/<id>/transitions | jq
```

## API一覧

| Method | Path | 説明 |
|---|---|---|
| POST | `/entities` | エンティティ作成（初期状態は `enrolled`） |
| GET | `/entities` | エンティティ一覧 |
| GET | `/entities/{id}` | エンティティ取得 |
| POST | `/entities/{id}/transitions` | 状態遷移の実行 |
| GET | `/entities/{id}/transitions` | 遷移履歴（監査ログ）の取得 |

## テスト

状態遷移の合法性ルール(`internal/identity`の`CanTransition`)のテストはDB不要でそのまま動く。

```bash
go test ./...
```

それ以外（Store・API・e2e）はPostgreSQLへの接続が必要で、`TEST_DATABASE_URL`が未設定の場合は自動的にskipされる。

```bash
docker compose up -d
TEST_DATABASE_URL="postgres://identity:identity@localhost:5432/identity_lifecycle?sslmode=disable" go test ./...
```

## 現状のスコープと今後の拡張ポイント

このリポジトリは「許可された状態遷移だけを通し、DBの仕組みで監査ログを記録し、永続化とグレースフルシャットダウンを備える」ところまでを実装した学習用の第二歩です。実運用のIdentity管理基盤として使うには、少なくとも以下が不足しています。

- **認可（誰が遷移を起こせるか）**: `actor` はリクエストボディの自己申告文字列で、なりすまし放題になっている。認証済みのユーザー/システムIDに紐付ける必要がある
- **外部トリガーとの連携**: 実際の失効・一時停止は多くの場合、他システムからのWebhookや有効期限バッチなど、イベント駆動で起きる。現状はAPI呼び出しに頼っている
- **入力検証・レート制限・認証**: APIとして最低限必要なガードレールは未実装
- **マイグレーション管理**: 現在はdocker初回起動時に`init.sql`を流すだけで、スキーマ変更を追跡する仕組み（golang-migrate等）がない
