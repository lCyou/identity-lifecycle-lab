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

- `internal/identity`: 状態定義・遷移ルール・エンティティ・監査ログのインメモリストア
- `internal/api`: REST APIハンドラ（標準ライブラリの `net/http` のみ使用）
- `main.go`: HTTPサーバー起動（`:8080`）

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
