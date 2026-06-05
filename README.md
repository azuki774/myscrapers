# myscrapers

## myscraper (Go)
- 新しい Go ベースの scraper 実装は `myscraper/` 配下にある
- 既存 Python scraper は `src/` 配下に残し、legacy 扱いで維持する
- 開発時は `nix develop` の上で `cd myscraper && go test ./...` を使う

### myscraper CLI
```bash
nix develop
cd myscraper
go test ./internal/... -v
PLAYWRIGHT_E2E=1 go test ./e2e -run TestGitHubSmoke -v
go run ./cmd/myscraper --url https://github.com --out tmp/github.html
```

### myscraper SMBC Card WEB明細 HTML dump

Current scope: authenticated login and raw HTML snapshot only. CSV parsing for the WEB明細 table comes later.

Environment variables:

- `SMBC_VPASS_ID` and `SMBC_VPASS_PASSWORD`
- or the legacy fallback pair `user` and `pass`

Run:

```bash
nix develop
cd myscraper
go test ./...
go run ./cmd/myscraper --mode smbc-card-webmeisai --out tmp/smbccard-webmeisai.html
```

Optional real-browser smoke test:

```bash
nix develop
cd myscraper
PLAYWRIGHT_E2E_SMBCCARD=1 go test ./e2e -run TestSMBCCardWebMeisaiSmoke -v
```

Chromium launch flags: chromium needs extra flags in some environments (running as root, inside a container, or in the nix dev shell where the SUID sandbox cannot start). Pass them through the `MYSCRAPER_CHROMIUM_ARGS` environment variable, whitespace-separated. Default is none.

```bash
MYSCRAPER_CHROMIUM_ARGS="--no-sandbox --disable-dev-shm-usage" \
  go run ./cmd/myscraper --mode smbc-card-webmeisai --out tmp/smbccard-webmeisai.html
```

User-Agent: the browser package overrides the default Playwright User-Agent with a hardcoded Chrome 148 / Linux x86_64 string, set on every `BrowserContext`. The intent is to avoid the `HeadlessChrome` UA that stock Playwright ships, which the Vpass Akamai front-end flags as automated. Pair with `--disable-blink-features=AutomationControlled` in `MYSCRAPER_CHROMIUM_ARGS` to also drop the `navigator.webdriver` flag, since the two signals are usually checked together.

```bash
MYSCRAPER_CHROMIUM_ARGS="--no-sandbox --disable-dev-shm-usage --disable-blink-features=AutomationControlled" \
  go run ./cmd/myscraper --mode smbc-card-webmeisai --out tmp/smbccard-webmeisai.html
```

Note: Vpass (and other Akamai / Cloudflare-fronted sites) may still 403 the request on TLS / HTTP/2 fingerprint grounds; a headless chromium cannot always be made to look like a real desktop browser, and a residential proxy or real Chrome in headed mode may be the only reliable bypass.

## myscrapers-sbi
- SBIのポートフォリオを保存
- https://site1.sbisec.co.jp/ETGate/ に自動的にログインして、ポートフォリオの表ごとに保存する。
- 出力先は、コンテナ内の /data/YYYYMMDD_x.csv
    - x: 連番
    - outputDir オプションがあった場合は、${outputDir}/YYYYMM/YYYYMMDD_x.csv
- s3ストレージにアップロードへの機能がある。
    - 環境変数 `BUCKET_NAME` があった場合、取得したデータを `s3://${BUCKET_NAME}/${REMOTE_DIR}/YYYYMM/YYYYMMDD/` に保存。

## myscraper-mf
- マネーフォワードの家計簿パートを保存
- 同時に、口座更新のボタンも押して、データを更新する
- 出力先は、コンテナ内の /data/cf.csv, /data/cf_lastmonth.csv
    - 今月分と先月分のCSVファイルを出力
    - 本家DL機能との差異は docs/ ディレクトリを参照
- s3ストレージにアップロードへの機能がある。
    - 環境変数 `BUCKET_NAME` があった場合、取得したデータを `s3://${BUCKET_NAME}/${REMOTE_DIR}/` に保存。

```
* データを取得し、CSVファイルとして保存する場合（これまでと同じ動作）

   $ python src/moneyforward/main.py fetch

   * データを取得し、S3にアップロードする場合

   $ python src/moneyforward/main.py fetch --s3-upload

   * 口座情報を更新する場合

   $ python src/moneyforward/main.py update

   * ヘルプを表示する場合

   $ python src/moneyforward/main.py --help
     これにより、利用可能なサブコマンドとオプションの一覧が表示されます。
```
