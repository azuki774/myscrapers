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

### myscraper-mf (Go)

The Go re-implementation of the MoneyForward scraper lives in `myscraper/`
and is built into the same `myscrapers-mf` container image. The Python
sources under `src/moneyforward/` are kept for legacy reference only.

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

### myscraper-mf local with podman compose

`deployment/compose.yml` mounts `deployment/cookie.json` into the container as
`/data/cookie.json` and writes scraper output to `deployment/out/`.
`deployment/cookie.json` is ignored by Git so the browser-exported cookie file
does not get committed by accident.

```bash
mkdir -p deployment/out
cp /path/to/browser-exported-cookie.json deployment/cookie.json
podman compose -f deployment/compose.yml build
podman compose -f deployment/compose.yml run --rm myscrapers-mf

# run update instead of fetch
podman compose -f deployment/compose.yml run --rm myscrapers-mf \
  moneyforward --update
```

`podman-compose` may resolve bind mounts relative to the current working
directory, so this file uses `./deployment/...` host paths and the commands
above should be run from the repository root.
