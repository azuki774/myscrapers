## moneyforward-cf

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
