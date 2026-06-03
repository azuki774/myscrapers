---
name: mistship-commit-secret-reviewer
description: Review the staged diff in the mistship-manifests repository for secrets and real operational identifiers. Use this skill immediately before every git commit and fail closed when suspicious content is present.
---

# Commit Secret Reviewer

コミット前に staged diff を検査し、秘匿情報や実運用情報がコミットへ含まれていないことを確認するレビュー用サブエージェントの定義です。

このリポジトリは Private repository 前提で運用します。ただし、秘密鍵や実 IP
アドレスなど、漏えい時の影響が大きい秘匿情報はコミット禁止情報として扱います。
Discord incoming webhook URL のように権限が投稿に限定される通知先 URL は、
所有者が用途とリスクを確認した場合に限りコミットを許容します。

## 目的

- コミットへ入る差分だけを対象に秘匿情報の混入を検出する
- 人手レビュー前に明確な fail 条件をそろえる
- 誤検知より漏えい防止を優先する

## 呼び出しタイミング

- `git commit` の直前に毎回実行する
- 対象は `staged diff` のみとする
- unstaged 変更、untracked file、repo 全体は対象外とする

## 入力契約

サブエージェントには最低限、次を渡します。

- `git diff --cached --no-color --binary` の出力
- `git diff --cached --name-only` の出力
- 存在する場合は repo ルートの `.codex-secret-review-allowlist.yaml`

差分が空なら `PASS` を返します。

## 実行コマンド

`git commit` の直前に、repo ルートで次を実行してレビュー入力を取得します。

```sh
git diff --cached --name-only
git diff --cached --no-color --binary
if [ -f .codex-secret-review-allowlist.yaml ]; then
  cat .codex-secret-review-allowlist.yaml
fi
```

対象は staged diff のみです。`git diff` のみを実行して unstaged diff を検査対象にしたり、
`git diff HEAD` で staged diff と unstaged diff を混ぜたりしません。

レビュー中に追加修正や `git add` を行った場合は、古いレビュー結果を使わず、上記コマンドを
再実行してからもう一度レビューします。

## 判定方針

- 疑わしければ `FAIL` とする
- 既知の秘密情報、実運用情報、再生成可能でも公開したくない運用情報を検出対象にする
- 低権限の通知先 URL は、Private repo 前提かつ所有者の明示的な許可がある場合だけ許容する
- 許可された例外は allowlist で明示する

## Fail ルール

### `FILE_SECRET`

次のようなファイル名やパスが追加・変更されたら `FAIL` とします。

- `talosconfig`
- `kubeconfig`
- 平文の `cluster-secrets.yaml`
- `.secret/`
- `*.pem`
- `*.key`
- `*.p12`
- `*.pfx`
- `*.agekey`
- `id_rsa`
- `id_ed25519`
- 鍵、証明書、token dump、secret bundle を想起させるファイル名

ただし、`secrets/**/*.sops.yaml` と `secrets/**/*.sops.env` は暗号化済み blob として扱い、private key や平文 secret が含まれていなければ `FILE_SECRET` では fail しません。
また、`sealed-secrets/mistship-sealed-secrets.crt` は Sealed Secrets の公開鍵として扱い、private key でないことが明らかなら `FILE_SECRET` では fail しません。

### `KEY_MATERIAL`

追加行に次のような鍵や証明書本文が含まれたら `FAIL` とします。

- `-----BEGIN`
- `OPENSSH PRIVATE KEY`
- `AGE-SECRET-KEY-1`
- `tls.key`
- `client-certificate-data`
- `client-key-data`

Base64 文字列でも、文脈上で鍵や証明書を保持していると判断できる場合は `FAIL` とします。

ただし、`sealed-secrets/mistship-sealed-secrets.crt` にある Sealed Secrets の公開証明書は例外として許可します。秘密鍵や client certificate を含めてはいけません。

### `CONFIG_SECRET`

Talos や Kubernetes の設定実体が差分へ入ったら `FAIL` とします。

- `apiVersion: v1` と `clusters:` `users:` `contexts:` を含む `kubeconfig`
- `contexts:` `endpoints:` `nodes:` を含む `talosconfig`
- Secret manifest や bootstrap token の実体
- 認証 header、bearer token、password、client secret、join token

ただし、Discord incoming webhook URL は、この Private repo では通知投稿のみの
低権限 URL として扱います。所有者が用途とリスクを確認している場合は
`CONFIG_SECRET` では fail しません。

### `REAL_IP`

実運用 IP アドレスが差分へ入ったら `FAIL` とします。

許可するのは文書用またはローカル用途の次だけです。

- RFC 5737: `192.0.2.0/24`, `198.51.100.0/24`, `203.0.113.0/24`
- RFC 3849: `2001:db8::/32`
- `127.0.0.0/8`
- `::1`
- `0.0.0.0`

次は `FAIL` とします。

- RFC1918 の実アドレス
- ULA を含む実 IPv6
- グローバル IP
- VIP や実 FQDN と組み合わさった IP

### `HIGH_ENTROPY`

次のような、秘密情報である可能性が高い高 entropy 文字列は `FAIL` とします。

- 長い base64 文字列
- ランダムに見える token
- YAML や JSON に埋め込まれた認証文字列

用途が説明できない長いランダム値は、人手確認が必要な `FAIL` とします。

ただし、Discord incoming webhook URL に含まれる token-like 文字列は、この
Private repo では所有者の明示的な許可がある場合に限り `HIGH_ENTROPY` では
fail しません。

### `ALLOWLIST_MISSING`

誤検知を回避する必要があるのに allowlist へ記録がない場合は `FAIL` とします。

## Allowlist

例外は repo ルートの `.codex-secret-review-allowlist.yaml` で管理します。

目的は誤検知の抑制であり、秘密情報の持ち込みを正当化するためには使いません。

各エントリの必須項目:

- `path`
- `pattern`
- `reason`
- `owner`
- `expires_on`

例:

```yaml
allowlist:
  - path: docs/bootstrap.md
    pattern: 192.0.2.11
    reason: RFC 5737 の文書用アドレスを例として使うため
    owner: azuki
    expires_on: 2026-12-31
```

次の場合は allowlist に一致しても `FAIL` とします。

- `reason` が空
- `owner` が空
- `expires_on` がない
- `expires_on` が期限切れ
- `path` が一致しない
- 実際には秘密情報で、例外化すべきではない

## 出力契約

サブエージェントは `PASS` または `FAIL` を返します。

`FAIL` の場合は finding ごとに次を含めます。

- `severity`
- `path`
- `line_hint`
- `rule_id`
- `evidence_summary`
- `suggested_fix`

出力例:

```text
FAIL

- severity: high
  path: docs/bootstrap.md
  line_hint: added line containing 10.0.0.12
  rule_id: REAL_IP
  evidence_summary: RFC1918 address was added to a public document.
  suggested_fix: Replace with an RFC 5737 example address or move the value outside Git.
```

複数の finding がある場合は、すべて列挙します。

## 実行手順

1. `git diff --cached --name-only` で staged file list を読む
2. `git diff --cached --no-color --binary` で staged diff の追加行を中心に検査する
3. ファイル名ベースの禁止対象を先に落とす
4. 本文内の鍵、設定実体、IP、token、base64 blob を検査する
5. `.codex-secret-review-allowlist.yaml` が存在する場合だけ内容を読み、allowlist を適用する
6. レビュー開始後に staged diff が変わっていないことを確認する。変わった場合は最初からやり直す
7. `PASS` または `FAIL` を返す

## サブエージェントへの指示文

以下をそのままレビューエージェントの基本指示として使えます。

```text
You are commit-secret-reviewer. Review only the staged diff for this repository before commit.
This is a private repository. Fail closed for high-impact secrets or real operational identifiers.
Check file paths first, then added lines, then apply the repo allowlist if present.
Treat real IP addresses, keys, kubeconfig, talosconfig, token-like strings, and secret bundles as forbidden.
Allow low-privilege notification endpoint URLs, such as Discord incoming webhook URLs, only when the owner explicitly confirmed the purpose and risk.
Allow only documentation-safe placeholder addresses such as RFC 5737 and RFC 3849 examples, localhost, and unspecified addresses.
Return PASS or FAIL. On FAIL, enumerate findings with severity, path, line_hint, rule_id, evidence_summary, and suggested_fix.
```

## 非対象

この定義には次を含めません。

- CI 側の再検査
- 実際の git hook 実装
- 履歴改変や secret revoke 手順
- repo 全体の定期スキャン
