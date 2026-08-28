# myscrapers コマンド結果ファイル 外部仕様書

本ドキュメントは Go 製 CLI `myscraper`（`myscraper/cmd/myscraper`）が各コマンド実行時に生成する**結果ファイル**の外部仕様を定義する。
入力ファイル（cookie.json・パスキー JSON）は対象外とする。

## 対象コマンドと結果ファイル一覧

| コマンド | 結果ファイル | 備考 |
|---|---|---|
| `myscraper moneyforward --fetch` | `cf.csv`, `cf_lastmonth.csv`, `asset_history.csv` | `--s3-upload` 併用時は S3 へ同一ファイル名でアップロード |
| `myscraper moneyforward --fetch`（CF テーブルパース失敗時） | `cf_now_debug.html`, `cf_lastmonth_debug.html` | 失敗時のみ生成されるデバッグ用スナップショット |
| `myscraper moneyforward --update` | （なし） | サイト上でボタン押下を行うのみ |
| `myscraper moneyforward fetch-cookie` | `--cookie-path` で指定した cookie.json | S3 からダウンロード。`/data/cookie.json` が既定 |
| `myscraper sbi [--output FILE]` | `assets.json`（`--output` 指定時）または stdout | パーミッション 0600 |
| `myscraper --url URL [--out FILE]` | HTML スナップショット | 汎用ページ取得。既定値 `tmp/page.html` |

## 共通仕様

- **文字エンコーディング**: 全結果ファイルとも UTF-8（BOM なし）。
  - 注: legacy Python 版（`src/moneyforward/main.py`）は CSV を Shift_JIS へ変換していたが、Go 実装では変換を行わないため、UTF-8 のまま書き出される。`docs/myscrapers.md` の Shift-JIS 記述は現行実装との乖離がある。
- **改行コード**: LF。
- **時刻**: `moneyforward` の日付変換、および `sbi` の `fetched_at` は実行時時刻（`time.Now()`）を使用する。

---

## 1. `moneyforward --fetch` の結果ファイル

### 1.1 `cf.csv` / `cf_lastmonth.csv`

現在月 / 先月のキャッシュフロー明細（CF）テーブルを書き出した CSV。

- **出力先**: `<output-dir>/cf.csv`, `<output-dir>/cf_lastmonth.csv`（`--output-dir` / `MF_OUTPUT_DIR` で変更可。既定 `/data`）
- **パーミッション**: 0644
- **ソース**: MoneyForward の `/cf` ページ内 `<table id="cf-detail-table">` のデータ行。可視テキストを持たない空行はスキップされる。
- **形式**: 固定 10 カラム・全フィールドがダブルクオートで囲まれた手書き CSV。ヘッダー行＋データ行で構成、各行 LF 終端。

ヘッダー行（固定文字列）:

```
"計算対象","日付","内容","金額（円）","保有金融機関","大項目","中項目","メモ","振替","ID"
```

データ行の変換規則（インデックス 0 始まり）:

| # | 列名 | 内容 |
|---|---|---|
| 0 | 計算対象 | 常に `1`（Python 版と同挙動。全件を対象として扱う） |
| 1 | 日付 | ページ上の `MM/DD(曜日)` を `YYYY/MM/DD` に変換。年は実行時時刻の年。ただし `cf_lastmonth.csv` で月が `12` の場合は年を 1 戻す（例: 1 月に前月＝昨年 12 月分を取得するケース） |
| 2 | 内容 | セルのテキストの**先頭行のみ**（セル内改行は切り捨て） |
| 3 | 金額（円） | セルのテキストの先頭行。ページ表記のまま（例: `-110`） |
| 4 | 保有金融機関 | セルのテキストの先頭行 |
| 5 | 大項目 | セルのテキストの先頭行 |
| 6 | 中項目 | セルのテキストの先頭行 |
| 7 | メモ | セルのテキストの先頭行 |
| 8 | 振替 | セルのテキストの先頭行 |
| 9 | ID | セルのテキストの先頭行 |

実装の制約（外部仕様として把握しておくこと）:

- 各フィールドは `fmt.Sprintf` による直書きのため、セル内容に `"`（ダブルクオート）が含まれる場合のエスケープ処理は行われない。RFC 4180 準拠ではない。
- 日付変換は `MM/DD` 形式を前提としており、形式違いの入力に対する検証・エラー処理はない。

サンプル:

```
"計算対象","日付","内容","金額（円）","保有金融機関","大項目","中項目","メモ","振替","ID"
"1","2026/08/22","物販","-110","モバイルSuica","未分類","未分類","","",""
"1","2026/08/23","コンビニ","-580","三井住友","食費","外食","メモ","",""
```

### 1.2 `asset_history.csv`

資産推移（`/bs/history`）テーブルを書き出した CSV。

- **出力先**: `<output-dir>/asset_history.csv`
- **パーミッション**: 0644（`os.Create` の既定）
- **ソース**: `/bs/history` ページ内 `<table class="table table-bordered">`。`thead` の最初の `tr` をヘッダー行、`tbody` の各 `tr` をデータ行とする（`thead`/`tbody` が無い場合はテーブル直下の全 `tr` を対象）。
- **形式**: Go `encoding/csv` Writer による出力。RFC 4180 に近い規則（カンマ・ダブルクオート・改行を含むフィールドのみ引用符で囲み、`"` は `""` に二重化。それ以外は無引用）。ヘッダー行＋データ行。

セルの変換規則:

- テキストの末尾が `円` のセルは末尾の `円` を除去して数値のみ出力（例: `1,234,567円` → `1,234,567`）。
- `<a>` リンクのテキストが `詳細` のセルは、リンクの代わりに文字列 `詳細` を出力。
- CF テーブルと異なり、空行のフィルタは行われない。

サンプル:

```
月,資産合計,詳細
2026-08,1234567,詳細
2026-07,1200000,詳細
```

### 1.3 デバッグ用 HTML（パース失敗時のみ）

CF テーブル（`#cf-detail-table`）のパースに失敗した場合、その時点のページ HTML 全体をスナップショットとして書き出す。

- **ファイル**: 現在月失敗時 `cf_now_debug.html` / 先月失敗時 `cf_lastmonth_debug.html`
- **出力先**: `<output-dir>/`（`--output-dir` / `MF_OUTPUT_DIR`）
- **パーミッション**: 0600
- 資産推移テーブルのパース失敗時にはデバッグ HTML は書き出されない。
- 処理はエラーで終了する（exit code 1）。

### 1.4 アップロード先（`--s3-upload` 併用時）

`--s3-upload` を付けた場合、上記 3 件の CSV をローカル書き出し後に S3 へアップロードし、成功すれば終了コード 0 を返す。

| オブジェクトキー |
|---|
| `<BUCKET_DIR>/cf.csv` |
| `<BUCKET_DIR>/cf_lastmonth.csv` |
| `<BUCKET_DIR>/asset_history.csv` |

- 必要な環境変数: `BUCKET_URL`, `BUCKET_NAME`, `BUCKET_DIR`, `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`。
- アップロードは 3 ファイルが順に実行され、いずれかが失敗するとエラーで終了する（exit code 1）。

---

## 2. `moneyforward --update` の結果ファイル

結果ファイルは生成しない。アカウント一覧ページの「一括更新」押下と、ホームページ上のモバイルSuica「更新」リンク押下（見つからない場合は警告ログのみ）を実行し、正常時は終了コード 0 を返す。

---

## 3. `moneyforward fetch-cookie` の結果ファイル

S3 から cookie ファイルをダウンロードしてローカルに保存する。

- **ファイル**: `--cookie-path` / `MF_COOKIE_PATH`（既定 `/data/cookie.json`）
- **取得元**: `s3://<BUCKET_NAME>/<BUCKET_DIR>/cookie.json`
- **内容**: ブラウザ拡張がエクスポートした cookie の JSON 配列。各要素は `name`, `value`, `domain`, `path`, `secure`, `httpOnly`, `expirationDate`, `sameSite` を持つ。このファイルは次回以降の `--fetch` / `--update` の入力として使用される。
- 必要環境変数は `--s3-upload` と同様。

---

## 4. `sbi` の結果ファイル（資産サマリー JSON）

保存済みパスキーでログインし、SBI 証券の資産残高を JSON で出力する。

- **出力先**: `--output` / `SBI_OUTPUT` で指定したファイル。未指定時は **stdout** へ出力。
- **パーミッション**: 0600（金融情報のため）
- **形式**: UTF-8、2 スペースインデントの JSON、末尾に改行 1 つ。

トップレベル構造:

```json
{
  "schema_version": 1,
  "fetched_at": "2026-08-16T12:00:00Z",
  "nisa": { ... },
  "old_nisa": { ... },
  "cash": { ... },
  "others": { ... },
  "grand_total_jpy": 1160000
}
```

各フィールドの意味:

| フィールド | 内容 |
|---|---|
| `schema_version` | 出力フォーマットのバージョン。常にトップレベルの最初のキーとして出力される（下記「スキーマバージョニング」参照） |
| `fetched_at` | 取得実行時刻（RFC 3339 / ISO 8601 形式） |
| `nisa` | 新 NISA ポートフォリオ全体（国内株式 + 米国株式 + 投資信託）。残高・前日比/率・前月比/率・評価損益/率と、資産クラス別内訳を持つ |
| `old_nisa` | 旧つみたてNISA 投資信託。残高・前日比/率・評価損益/率と銘柄別内訳を持つ（前月比は SBI 側で表示が無いため存在しない） |
| `cash` | 預り金残高。通貨別（`jpy` / `usd`）の金額エントリを持つ（下記「金額エントリ」参照） |
| `others` | nisa / old_nisa / cash 以外。現在は特定預り投資信託（`funds`）のみ。将来のセクション追加はここに行われる |
| `grand_total_jpy` | 全資産合計（円）。下記 MECE 合計と一致 |

**金額エントリ**: `cash` / `others` 配下の各項目は、必ず次の 2 フィールドを持つオブジェクトである。

| フィールド | 内容 |
|---|---|
| `amount` | 自通貨建ての数量（JPY エントリなら円、USD エントリなら米ドル） |
| `value_jpy` | 円換算評価額。JPY エントリでは `amount` と同値になる |

取り込み側は条件分岐なしにすべての `value_jpy` を合計すればよい。

`grand_total_jpy` の計算式（MECE: 重複なし・漏れなし）:

```
grand_total_jpy =
  nisa.total_jpy
  + old_nisa.total_jpy
  + cash.jpy.value_jpy
  + cash.usd.value_jpy
  + others.funds.value_jpy
```

### 4.1 `nisa` 詳細

| フィールド | 内容 |
|---|---|
| `total_jpy`, `prev_day_jpy`, `prev_day_pct`, `prev_month_jpy`, `prev_month_pct`, `pnl_jpy`, `pnl_pct` | 口座全体の残高・前日比・前月比・評価損益（金額は円、率は %） |
| `domestic_stocks` / `us_stocks` / `funds` | 国内株式 / 米国株式 / 投資信託のクラス別内訳 |

各クラス内訳（`domestic_stocks` 等）:

| フィールド | 内容 |
|---|---|
| `value_jpy` | クラスの評価額（円） |
| `pnl_jpy`, `pnl_pct` | 評価損益（円/%） |
| `prev_day_jpy`, `prev_day_pct`, `prev_month_jpy`, `prev_month_pct` | 前日比・前月比（円/%） |
| `holdings` | 銘柄別明細の配列 |

銘柄明細 `holdings[]` / `old_nisa.funds[]` の各要素:

| フィールド | 内容 |
|---|---|
| `name` | 銘柄名 |
| `quantity` | 保有数量（投資信託は口数） |
| `unit_cost` | 取得単価（米国株式は USD、投資信託は 1 万口あたり） |
| `unit_price` | 現在値（米国株式は USD、投資信託は 1 万口あたり） |
| `prev_day_jpy`, `prev_day_pct` | 前日比（円/%）。米国株式には前日比情報が無いため 0 になる |
| `pnl_jpy`, `pnl_pct` | 評価損益（円 / %。株・投信は円建て、米国株式の P&L は円換算） |
| `value_jpy` | 評価額（円） |

### 4.2 `old_nisa` 詳細

| フィールド | 内容 |
|---|---|
| `total_jpy`, `prev_day_jpy`, `prev_day_pct`, `pnl_jpy`, `pnl_pct` | 口座全体の残高・前日比・評価損益 |
| `funds` | 銘柄別明細の配列（上部の銘柄明細と同じ形式） |

### 4.3 `cash` / `others` 詳細

`cash`:

| フィールド | 内容 |
|---|---|
| `jpy` | 国内現金残高等。金額エントリ（`amount` = `value_jpy` = 円額） |
| `usd` | 米ドル預り金。金額エントリ（`amount` = 米ドル数量、`value_jpy` = 円換算評価額） |

`others`:

| フィールド | 内容 |
|---|---|
| `funds` | 特定預りの投資信託。金額エントリ（円） |

将来 `others` に新しいセクションキーを追加する場合は後方互換変更とし、`schema_version` は上げない。

 完全なサンプルは `myscraper/internal/sbi/testdata/example-assets.json` を参照（`go test` で構造・`schema_version` と MECE 整合性が検証される）。

### 4.4 S3 アップロード（`--s3-upload`）

`--s3-upload` を指定すると、**入力のパスキーも S3 から取得**し、stdout/ファイルへ出力した JSON と同じバイト列を S3 にも保存する。S3 モードではローカルの `--passkey` / `SBI_PASSKEY_PATH` は無視される（警告ログを出力）。必要環境変数は MoneyForward の `--s3-upload` と共通（`BUCKET_URL`, `BUCKET_NAME`, `BUCKET_DIR`, `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`）。

- パスキー取得元: `s3://<BUCKET_NAME>/<BUCKET_DIR>/passkey.json`。ダウンロードは一時ファイル（0600）へ行い、検証後にログインへ渡す。処理終了時に一時ファイルは削除される。
- オブジェクトキー（結果）: `BUCKET_DIR/YYYY/MM/YYYYMMDD-HHMMSS.json`（取得時刻を JST に変換）。実行ごとに履歴として残る。
- Content-Type: `application/json`
- 保存タイミング: 先に stdout/ファイルへ出力した後、S3 へアップロードする。
- メンテナンス時: `status: "maintenance"` を含む部分的な JSON も、通常の成功結果として同様に保存する。
- 失敗時:
  - S3 環境変数の不足、パスキーのダウンロード失敗、ダウンロードしたパスキーの検証失敗はいずれも終了コード 1（ログイン・取得は行わない）。
  - S3 アップロード失敗時は、それまでの stdout/ファイル出力は残したままコマンドは終了コード 1 になる。

`--s3-upload` を付けないローカルモードでは、従来どおり `--passkey` / `SBI_PASSKEY_PATH` / 既定ローカルパスからパスキーを読み、S3 へは保存しない。

### 4.5 スキーマバージョニング

- `schema_version` は 1 始まりの整数。現在のバージョンは **1**（実装上の単一の定数 `sbi.CurrentSchemaVersion`）。書き出し時に必ず付与されるため、stdout 出力・ファイル出力のどちらでも常に先頭キーとして現れる。
- **バージョンを上げる（破壊的変更）**: フィールドの削除・リネーム、フィールドの型変更、既存フィールドの値の意味変更など、旧バージョンを前提とした取り込み処理が壊れる変更。
- **バージョンを上げない（後方互換変更）**: 新しいフィールドの追加、`status` のような文字列列挙値への新しい値の追加など、読み飛ばしても取り込み処理が壊れない変更。
- 取り込み側システムは、未知の `schema_version` をエラーとして扱うことを推奨する。これにより、本ドキュメントの更新漏れがあっても誤った解釈での取り込みを防げる。

---

## 5. `--url`（汎用スクレイプ）の結果ファイル

任意の URL をブラウザで取得し、描画後の HTML スナップショットをファイルに書き出す。

- **出力先**: `--out` で指定（既定 `tmp/page.html`）
- **パーミッション**: 0644
- **内容**: レンダリング完了後の完全な HTML。親ディレクトリが無い場合は 0755 で自動作成される。
- ログにはページタイトルと最終 URL（リダイレクト後）が出力される。
- 認証済み金融ページの HTML など機密性の高い取得物を扱う場合は、この既定パーミッション（0644）に注意して別パスを利用すること。