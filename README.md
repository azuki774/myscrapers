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
