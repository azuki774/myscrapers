# myscrapers

## Container images

The `v*` release workflow publishes one image for `linux/amd64` with semver
tags and `latest`. Arm64 publishing is temporarily suspended because the
additional build time is too high:

- `ghcr.io/azuki774/myscrapers`

Pushes to `master` publish the image with the first seven characters of the
commit SHA as the tag (for example, `a1b2c3d`). These pushes do not update
`latest`. Pushes to other branches build the image without publishing it.
Existing tags for the previous three image names remain available for rollback,
but the workflow no longer updates them.

The image entrypoint is `myscraper`. Its default command is
`moneyforward --fetch --s3-upload`; pass `sbi` or `nrkn` and their flags as
arguments when running those scrapers.

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

### myscraper nrkn CLI

NRKN の「資産評価額照会 → 合計」から全商品の最新明細を取得します。
実行環境から `NRKN_ID`、`NRKN_PASS`、`NRKN_BIRTHDAY`（YYYYMMDD）を注入してください。
認証情報を引数やログに記載しないでください。

```bash
# 開発環境で実行。出力先ディレクトリは事前に用意する。
go run ./cmd/myscraper nrkn --output /data/nrkn-assets.json
# S3設定はSBIと共通。BUCKET_DIRは myscrapers/nrkn を推奨。
go run ./cmd/myscraper nrkn --s3-upload
```

`--output`（未指定時 `NRKN_OUTPUT`）がなければ stdout にJSONのみを出力します。
ログはstderr、ファイルは0600で保存します。S3では実行時刻をJSTに変換した
`<BUCKET_DIR>/YYYY/MM/YYYYMMDD-HHMMSS.json` に同じ内容を保存します。
商品コード、名称、分類、数量、基準価額、評価額、取得価額累計、解約価額、
解約時評価額、損益、基準日、資産比率を収録します。詳細は [結果ファイル仕様](docs/result-files.md) を参照してください。
架空データの全項目例は [example-assets.json](myscraper/internal/nrkn/testdata/example-assets.json) にあります。

各商品には `composite_figi` を必ず付与します。SBIと共通のOpenFIGIクライアントで、
投信協会コードを `TICKER`、市場を `JP` として `/v3/mapping` へ問い合わせます。
APIキー不要、重複を除いて最大10件ずつ送信し、429/5xxのみ最大3回（初回を含む）、
`Retry-After` に従って再試行します。これはOpenFIGIへの再試行であり、NRKNへの再ログインは増やしません。

NRKNの5桁の商品コードはOpenFIGIへ送信しません。
[商品識別子一覧](myscraper/internal/nrkn/figi.go)で表示名を投信協会コードに対応付けます。
全半角・空白・英字大小の違いだけを正規化し、確認済み名称と完全一致させます。
現時点で登録済みなのはDCニッセイ国内株式インデックス、
野村外国株式インデックスファンド・MSCI-KOKUSAI（確定拠出年金向け）、
マイバランス70（確定拠出年金向け）の3商品です。
出典は[ニッセイの商品情報](https://www.nam.co.jp/fundinfo/dcnkki/main.html)と
[野村の投信協会コード一覧（3ページ）](https://www.nomura-am.co.jp/news/20160928_1D8BFFA4.pdf)です。
新しい商品は運用会社の資料でコードとNRKN表示名を確認して一覧・テストへ追加します。
FIGI自体は固定保存せず毎回問い合わせます。

未登録商品、複数候補、照合不一致、空のFIGI、APIエラーが1件でもあれば、
SBI同様に新しいJSONを出力・保存・S3送信せず失敗とします（既存ファイルは保持）。
その場合もログアウトを1回試みます。

ローカルの模擬ページを使うブラウザテストは、開発環境で
`NRKN_BROWSER_TEST=1 go test -v ./internal/nrkn -run TestBrowserFlow` を実行します。

日次ジョブの例（cronのタイムゾーンをAsia/Tokyoに設定済みのホスト）:

```cron
15 7 * * * /usr/bin/flock -n /var/lock/nrkn-assets.lock /usr/bin/docker run --rm --env-file /etc/myscrapers/nrkn.env ghcr.io/azuki774/myscrapers:latest nrkn --s3-upload
```

環境変数ファイルは管理者が0600で配置し、NRKN認証情報とS3設定を指定します。
コンテナの既定コマンドは `nrkn --s3-upload` です。同時実行を避け、失敗後の自動再実行は設定しません。
休日も取得でき、取得日と商品基準日は異なることがあります。
ログインは通常1回、同時利用エラー990003の場合だけ同一セッションで追加1回です。
終了時はログアウトをクリックし、完了を確認します。取得後のログアウト失敗は
保存済みJSONを保持したまま非ゼロ終了で通知します。

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

S3 へのアップロードも `--s3-upload` で行えます。S3 モードでは**パスキーも S3 から取得**するため、`--passkey` は不要になります。必須環境変数は MoneyForward と共通(`BUCKET_URL`, `BUCKET_NAME`, `BUCKET_DIR`, `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)です。

```bash
export BUCKET_URL=https://...
export BUCKET_NAME=my-bucket
export BUCKET_DIR=myscrapers/sbi
export AWS_REGION=auto
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
go run ./cmd/myscraper sbi --s3-upload
# 1. s3://my-bucket/myscrapers/sbi/passkey.json をダウンロードしてログイン
# 2. 取得 JSON を s3://my-bucket/myscrapers/sbi/YYYY/MM/YYYYMMDD-HHMMSS.json に保存(JST)
```

- パスキーは `BUCKET_DIR/passkey.json` から一時ファイル(0600)へダウンロードし、検証後にログインへ渡します。`--passkey` を指定しても S3 モードでは無視されます。
- 結果キーは取得時刻を JST に変換した `BUCKET_DIR/YYYY/MM/YYYYMMDD-HHMMSS.json` で、実行ごとに履歴として残ります。
- Content-Type は `application/json` です。
- `status: "maintenance"` を含む部分的な JSON も、FIGI を解決できた場合は通常の成功結果として同様に保存します。
- パスキーのダウンロード失敗・検証失敗、または結果の S3 アップロード失敗時は、コマンドは終了コード 1 になります。

銘柄明細には `composite_figi`（OpenFIGI の composite FIGI）を必ず含めます。国内株・ETF は SBI の 4 文字証券コード、米国株は ticker と市場、投資信託は保有一覧 HTML の銘柄リンク（`path=fund/detail/<code>` または `fund_sec_code`） を照合元として OpenFIGI に問い合わせます。照合できない銘柄がある場合は JSON を出力せず、取得を失敗扱いにします。照合元の SBI 固有コードは JSON には出力しません。

ローカルでの実行（S3 を使わない）の場合は、パスキーの指定は `--passkey` フラグが最優先で、省略時は環境変数 `SBI_PASSKEY_PATH`、それも無ければデフォルト `~/.local/state/opencode/sbi-passkey.json` を使います:

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


## myscrapers (Go)

マネーフォワードの家計簿パートを保存する Go ベースの scraper。
実装は `myscraper/` 配下にある。

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
`deployment/out/`. The Playwright driver is included in the image, so no
separate driver volume is needed.
`deployment/cookie.json` is ignored by Git so the browser-exported cookie file
does not get committed by accident.

```bash
mkdir -p deployment/out
cp /path/to/browser-exported-cookie.json deployment/cookie.json
podman compose -f deployment/compose.yml build
podman compose -f deployment/compose.yml run --rm myscrapers
# run update instead of fetch
podman compose -f deployment/compose.yml run --rm myscrapers \
  moneyforward --update
```

If an older compose setup created `deployment/deployment/` or turned
`deployment/cookie.json` into a directory, remove those leftovers before
running the container again.
