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

## 構成

```
internal/
  identity/
    entity.go       # Entity モデル
    transition.go   # TransitionRecord（監査ログ）モデル
    state.go        # State と許可遷移ルール
    store.go        # インメモリストア（エンティティ + 監査ログ）
    id.go
  api/
    router.go       # ルーティング定義
    handlers.go      # HTTPハンドラ
test/
  e2e/
    lifecycle_test.go  # 実HTTPサーバー越しの全ライフサイクル結合テスト
main.go              # HTTPサーバー起動（:8080）
```

単体テストは Go の言語仕様上、対象パッケージと同じディレクトリにしか置けませんが、`internal/identity` `internal/api` 配下のテストはいずれも外部テストパッケージ（`identity_test` / `api_test`）にしており、公開APIのみに依存する形で実装コードとは明確に切り離しています。HTTP全体を通した結合テストは `test/e2e` に独立して置いています。

## 動かし方

```bash
go run .
```

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

```bash
go test ./...
```

## 現状のスコープと今後の拡張ポイント

このリポジトリは「許可された状態遷移だけを通し、遷移のたびに誰が・いつ・なぜを記録する」という状態機械の中核部分のみを実装した学習用の第一歩です。実運用のIdentity管理基盤として使うには、少なくとも以下が不足しています。

- **永続化**: 現在はプロセスメモリ上のみで、再起動すると監査ログごと消える。SQLite等でエンティティ・遷移履歴を永続化する必要がある
- **認可（誰が遷移を起こせるか）**: `actor` はリクエストボディの自己申告文字列で、なりすまし放題になっている。認証済みのユーザー/システムIDに紐付ける必要がある
- **外部トリガーとの連携**: 実際の失効・一時停止は多くの場合、他システムからのWebhookや有効期限バッチなど、イベント駆動で起きる。現状はAPI呼び出しに頼っている
- **並行制御**: 同一エンティティへの同時遷移リクエストに対する楽観ロック等は未実装
- **入力検証・レート制限・認証**: APIとして最低限必要なガードレールは未実装

学習ステップとしては、まず永続化を入れて監査ログの価値（後から「いつ・誰が・なぜ」を追える）を実感し、その後に認可・イベント駆動の仕組みを足していくのがおすすめです。
