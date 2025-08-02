# myscrapers

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
