# myscrapers

## myscraper (Go)

Go ベースの scraper 実装は `myscraper/` 配下にある。
開発時は `nix develop` の上で `cd myscraper && go test ./...` を使う。

### myscraper CLI

```bash
nix develop
cd myscraper
go test ./internal/... -v
PLAYWRIGHT_E2E=1 go test ./e2e -run TestGitHubSmoke -v
go run ./cmd/myscraper --url https://github.com --out tmp/github.html
```

### myscraper sbi CLI

SBI証券の資産サマリーを、保存済みパスキーで自動ログインして取得する。

```bash
nix develop
cd myscraper
go run ./cmd/myscraper sbi --passkey /home/azuki/.local/state/opencode/sbi-passkey.json
# → JSON(domestic 口座サマリー / 外国株式 NISA 保有 / 外貨預り金 / 合計)を stdout に出力

# ファイル出力
go run ./cmd/myscraper sbi --passkey ~/.local/state/opencode/sbi-passkey.json \
  --output ./out/assets.json
```

S3 へのアップロードも `--s3-upload` で行えます。有効化すると、stdout/ファイルへ出力した JSON と同じ内容を S3 に保存します。必須環境変数は MoneyForward と共通(`BUCKET_URL`, `BUCKET_NAME`, `BUCKET_DIR`, `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)です。

```bash
export BUCKET_URL=https://...
export BUCKET_NAME=my-bucket
export BUCKET_DIR=myscrapers/sbi
export AWS_REGION=auto
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
go run ./cmd/myscraper sbi --s3-upload
# → 取得 JSON を s3://my-bucket/myscrapers/sbi/YYYYMM/YYYYMMDD-HHMMSS.json に保存(JST)
```

- キーは取得時刻を JST に変換した `BUCKET_DIR/YYYYMM/YYYYMMDD-HHMMSS.json` で、実行ごとに履歴として残ります。
- Content-Type は `application/json` です。
- `status: "maintenance"` を含む部分的な JSON も、通常の成功結果として同様に保存します。
- S3 アップロードに失敗しても、それまでの stdout/ファイル出力は残り、コマンドは終了コード 1 になります。

パスキーの指定は `--passkey` フラグが最優先で、省略時は環境変数 `SBI_PASSKEY_PATH`、それも無ければデフォルト `~/.local/state/opencode/sbi-passkey.json` を使います:

```bash
export SBI_PASSKEY_PATH=~/.local/state/opencode/sbi-passkey.json
go run ./cmd/myscraper sbi
```

出力例(値はダミー。実際の口座の金額とは無関係)。全フィールドを含む完全な JSON は `myscraper/internal/sbi/testdata/example-assets.json` にあり、`go test` で構造・整合性(holdings の合計とクラス集計の一致、MECE 合計)が検証されます。

```json
{
  "fetched_at": "2026-08-16T11:46:51.908856153Z",
  "status": "ok",
  "nisa": {
    "total_jpy": 3771642,
    "prev_day_jpy": 0,
    "prev_day_pct": 0,
    "prev_month_jpy": 967383,
    "prev_month_pct": 34.49,
    "pnl_jpy": 697759,
    "pnl_pct": 22.69,
    "domestic_stocks": { "value_jpy": 238635, "pnl_jpy": -2140, "pnl_pct": -0.88, "prev_day_jpy": 0, "prev_day_pct": 0, "prev_month_jpy": 10115, "prev_month_pct": 4.42,
      "holdings": [ { "name": "日本製鉄", "quantity": 200, "unit_cost": 598, "unit_price": 674.4, "prev_day_jpy": -1.9, "prev_day_pct": -0.28, "pnl_jpy": 15280, "pnl_pct": 12.78, "value_jpy": 134880 } ] },
    "us_stocks":       { "value_jpy": 1396796, "pnl_jpy": 235728, "pnl_pct": 20.3, "prev_day_jpy": 0, "prev_day_pct": 0, "prev_month_jpy": 791050, "prev_month_pct": 130.59,
      "holdings": [ { "name": "アドバンスト マイクロ デバイシズ", "quantity": 5, "unit_cost": 201.05, "unit_price": 514.39, "pnl_jpy": 250529, "value_jpy": 409814 } ] },
    "funds":           { "value_jpy": 2136211, "pnl_jpy": 464171, "pnl_pct": 27.76, "prev_day_jpy": 0, "prev_day_pct": 0, "prev_month_jpy": 166218, "prev_month_pct": 8.43,
      "holdings": [ { "name": "ｅＭＡＸＩＳ Ｓｌｉｍ 新興国株式インデックス", "quantity": 223327, "unit_cost": 19333, "unit_price": 27337, "prev_day_jpy": 237, "prev_day_pct": 0.87, "pnl_jpy": 178750.93, "pnl_pct": 41.4, "value_jpy": 610509.01 } ] }
  },
  "old_nisa": {
    "total_jpy": 1383248.56,
    "prev_day_jpy": 7486.78,
    "prev_day_pct": 0.54,
    "pnl_jpy": 650638.94,
    "pnl_pct": 88.81,
    "funds": [
      { "name": "ｅＭＡＸＩＳ Ｓｌｉｍ 新興国株式インデックス", "quantity": 131210, "unit_cost": 12888, "unit_price": 27337, "prev_day_jpy": 237, "prev_day_pct": 0.87, "pnl_jpy": 189585.32, "pnl_pct": 112.11, "value_jpy": 358688.77 },
      { "name": "ひふみプラス", "quantity": 11544, "unit_cost": 47384, "unit_price": 83470, "prev_day_jpy": 359, "prev_day_pct": 0.43, "pnl_jpy": 41657.67, "pnl_pct": 76.16, "value_jpy": 96357.76 }
    ]
  },
  "other": {
    "cash_jpy": 24870,
    "funds_jpy": 81423.47,
    "usd_cash": { "usd": 879.3, "jpy": 140107 }
  },
  "grand_total_jpy": 5401291.03
}
```

構成は MECE(重複なし・漏れなし)で、`grand_total_jpy` は以下の合計です:

- `status`: 取得状態。通常は `"ok"`。SBI がメンテナンスページを返した場合(NISA 等のサービス停止時)は `"maintenance"` になり、該当セクションは空になり、残りの取れるページは引き続き取得します。取得全体は失敗せず JSON を出力します。
- `nisa`: 新NISA ポートフォリオ(国内株式 + 米国株式 + 投資信託)。前日比・前月比・評価損益付き。各クラスには `holdings`(銘柄別)あり。出典: NISA ポートフォリオページ + ポートフォリオページ + 外国株式保有銘柄ページ
- `old_nisa`: 旧つみたてNISA 投資信託。前日比・評価損益・銘柄別 `funds` 付き(前月比は SBI に表示がない)。出典: ポートフォリオページ
- `other`: 現金残高 + 特定預り投資信託 + 米ドル預り金。出典: 口座サマリー / ポートフォリオページ / 外国株式口座サマリー
- `grand_total_jpy` = `nisa.total_jpy + old_nisa.total_jpy + other.cash_jpy + other.funds_jpy + other.usd_cash.jpy`

- ログインは WebAuthn 仮想認証器に保存鍵を復元して行う(パスキー自動ログイン)
- UA は通常ブラウザを偽装している(SBI は HeadlessChrome をブロックするため)
- 取得ページは固定 URL 3 件のみ。LLM は使わない


## myscrapers-sbi (Python)

SBIのポートフォリオを保存する Python ベースの scraper。
実装は `src/sbi/` 配下にある。

- https://site1.sbisec.co.jp/ETGate/ に自動的にログインして、ポートフォリオの表ごとに保存する。
- 出力先は、コンテナ内の /data/YYYYMMDD_x.csv
    - x: 連番
    - outputDir オプションがあった場合は、${outputDir}/YYYYMM/YYYYMMDD_x.csv
- s3ストレージにアップロードへの機能がある。
    - 環境変数 `BUCKET_NAME`があった場合、取得したデータを `s3://${BUCKET_NAME}/${REMOTE_DIR}/YYYYMM/YYYYMMDD/` に保存。

## myscrapers (Go)

マネーフォワードの家計簿パートを保存する Go ベースの scraper。
実装は `myscraper/` 配下にある。

Python 版の実装は `src/moneyforward/` に残している（legacy 扱い）。

- 同時に、口座更新のボタンも押して、データを更新する
- 出力先は、コンテナ内の /data/cf.csv, /data/cf_lastmonth.csv, /data/asset_history.csv
    - 今月分と先月分のCSVファイル、および資産推移を出力
- s3ストレージにアップロードへの機能がある。
    - 環境変数 `BUCKET_*` と `AWS_*` を設定することで、S3にアップロードする。
- 詳細な使い方は `docs/myscrapers.md` を参照。

```bash
nix develop
cd myscraper
go test ./internal/... -v
MF_E2E=1 go test ./e2e -run TestMoneyforwardSmoke -v

# scrape CF + asset history and write /data/cf.csv, /data/cf_lastmonth.csv,
# /data/asset_history.csv
myscraper moneyforward --fetch

# same, with S3 upload
myscraper moneyforward --fetch --s3-upload

# press 一括更新 and モバイルSuica 更新
myscraper moneyforward --update

# override defaults
myscraper moneyforward --fetch --output-dir ./out --cookie-path ./cookie.json
```

### myscraper local with podman compose

`deployment/compose.yml` is stored inside `deployment/`, so it mounts `.`
(the compose file directory itself) to `/data`. That keeps the cookie at
`deployment/cookie.json` while still letting the scraper read
`/data/cookie.json`, `MF_OUTPUT_DIR=/data/out` keeps scrape output under
`deployment/out/`, and `PLAYWRIGHT_DRIVER_PATH=/data/.playwright-driver`
persists the Playwright driver across local runs.
`deployment/cookie.json` is ignored by Git so the browser-exported cookie file
does not get committed by accident.

```bash
mkdir -p deployment/out
cp /path/to/browser-exported-cookie.json deployment/cookie.json
podman compose -f deployment/compose.yml build
podman compose -f deployment/compose.yml run --rm myscrapers
# first run may download the Playwright driver into deployment/.playwright-driver

# run update instead of fetch
podman compose -f deployment/compose.yml run --rm myscrapers \
  moneyforward --update
```

If an older compose setup created `deployment/deployment/` or turned
`deployment/cookie.json` into a directory, remove those leftovers before
running the container again.
