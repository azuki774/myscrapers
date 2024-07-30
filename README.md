# myscrapers

### myscrapers download sbi
- SBIのポートフォリオを保存
- https://site1.sbisec.co.jp/ETGate/ に自動的にログインして、ポートフォリオの表ごとに保存する。
- 出力先は、コンテナ内の /data/YYYYMMDD_x.csv
    - x: 連番
    - outputDir オプションがあった場合は、${outputDir}/YYYYMMDD_x.csv
- s3ストレージにアップロードへの機能がある。
    - 環境変数 `BUCKET_NAME` があった場合、取得したデータを `s3://${BUCKET_NAME}/${REMOTE_DIR}/YYYYMMDD/` に保存。

## Quick start (binary)

```
docker run --rm -p 7327:7327 ghcr.io/go-rod/rod:v0.116.2
```

```
user=<your id> \
pass=<your pass> \
outputDir="." \
wsAddr="localhost:7327" `
build/bin/myscrapers download moneyforward
```
