# myscrapers

## myscrapers-sbi
- SBIのポートフォリオを保存
- https://site1.sbisec.co.jp/ETGate/ に自動的にログインして、ポートフォリオの表ごとに保存する。
- 出力先は、コンテナ内の /data/YYYYMMDD_x.csv
    - x: 連番
    - outputDir オプションがあった場合は、${outputDir}/YYYYMMDD_x.csv
- s3ストレージにアップロードへの機能がある。
    - 環境変数 `BUCKET_NAME` があった場合、取得したデータを `s3://${BUCKET_NAME}/${REMOTE_DIR}/YYYYMMDD/` に保存。


#### go-rod製のものは使わなくしました(no maintenance)

<details>

## myscrapers download sbi
- SBIのポートフォリオを保存
- https://site1.sbisec.co.jp/ETGate/ に自動的にログインして、ポートフォリオの表ごとに保存する。
- 出力先は、コンテナ内の /data/YYYYMMDD_x.csv
    - x: 連番
    - outputDir オプションがあった場合は、${outputDir}/YYYYMMDD_x.csv
- s3ストレージにアップロードへの機能がある。
    - 環境変数 `BUCKET_NAME` があった場合、取得したデータを `s3://${BUCKET_NAME}/${REMOTE_DIR}/YYYYMMDD/` に保存。

## myscrapers download moneyforward

### output CSV
例：
```
計算対象,日付,内容,金額（円）,保有金融機関,大項目,中項目,メモ,振替,削除
,07/16(火),ローソン,-291,三井住友カード,食費,食料品,,,
,07/16(火),GITHUB,-158,JCBカード,通信費,情報サービス,,,
,07/10(水),マクドナルド,-600,三井住友カード,食費,外食,,,
```

### 出力先
- コンテナ内デフォルト: `/data/YYYYMM/YYYYMMDD/cf.csv`, `--lastmonth` 付与時は、`/data/YYYYMM/YYYYMMDD/cf_lastmonth.csv` も出力。

## Quick start (binary)

```
docker run --rm -p 7327:7327 ghcr.io/go-rod/rod:v0.116.2
```

```
user=<your id> \
pass=<your pass> \
outputDir="." \
wsAddr="localhost:7327"
build/bin/myscrapers download moneyforward
```

</details>
