---
name: mistship-conventional-commit-writer
description: Generate a Conventional Commits message for changes in the mistship-manifests repository. Use this skill immediately before creating a git commit or when the user asks for a commit message.
---

# Conventional Commit Writer

AI がこのリポジトリ向けに Conventional Commits 形式の commit message を作るための指示書です。

目的は、差分の意図が短く一貫して伝わる commit message を安定して生成することです。

## 目的

- commit message の形式を repo 全体でそろえる
- 変更内容ではなく変更の意図を件名へ載せる
- AI が曖昧な subject や不適切な type を選ぶことを防ぐ

## 出力対象

AI は、基本的に次のどちらかを返します。

- 1 行の Conventional Commit 件名
- 必要な場合だけ本文つきの commit message

通常は 1 行で十分です。

## 基本形式

形式は次を使います。

```text
<type>(<scope>): <subject>
```

scope が不要なら次を使います。

```text
<type>: <subject>
```

破壊的変更なら `!` を付けます。

```text
<type>(<scope>)!: <subject>
```

## Type ルール

この repo では次を使います。

- `feat`: 新しい利用者向け機能や新しい運用手順を追加した
- `fix`: 壊れていた挙動や分かりづらい手順を修正した
- `docs`: ドキュメントだけを変更した
- `refactor`: 挙動を変えずに構造や配置を整理した
- `ci`: GitHub Actions や CI 用検証を変更した
- `test`: テストや検証用 fixture を追加、更新した
- `build`: build 入力や生成に関わる仕組みを変更した
- `chore`: 保守作業や雑多な更新で、`feat` `fix` `refactor` ほど意味が強くない
- `revert`: 既存 commit を取り消した

迷ったときは、変更したファイル種別ではなく、利用者や開発者から見た影響で type を選びます。

## Scope ルール

scope は必須ではありません。あると差分の主対象が明確になる場合だけ使います。

この repo では次を優先候補にします。

- `nix`: `flake.nix` や dev shell
- `docs`: `README.md` や `docs/`
- `scripts`: `scripts/`
- `bootstrap`: `manifests/bootstrap/` や bootstrap 手順
- `infra`: `manifests/infra/`
- `patches`: `patches/`
- `secrets`: `secrets/` や secret 運用文書
- `templates`: `templates/`
- `ci`: `.github/workflows/`
- `image`: `image.yml`
- `repo`: repo 全体の整理や rename

単一の明確な対象がない場合は scope を省略します。

## Subject ルール

- 英語で書く
- 先頭は動詞で始める
- 命令形または現在形で短く書く
- 末尾に `.` を付けない
- 冗長な語を避ける
- 何を変えたかより、なぜその変更が必要かが分かる表現を優先する

悪い例:

- `docs: update README`
- `fix: fix kubectl`
- `chore: miscellaneous changes`

良い例:

- `docs: simplify bootstrap documentation`
- `fix(nix): prefer repo kubeconfig in dev shell`
- `ci: replace cluster-operating workflows with bootstrap checks`

## Type の選び方

### `docs`

次のときは `docs` を使います。

- Markdown やコメントだけを変えた
- 手順の説明、例、表現を直した
- AI 向け instruction 文書を追加した

ただし、documented behavior ではなく実際の挙動が変わるなら `docs` ではなく `fix` や `feat` を使います。

### `fix`

次のときは `fix` を使います。

- 既存の手順が失敗していた
- 既存 script や shell 環境が期待どおり動かなかった
- 既存 manifest や patch の不整合を直した

### `feat`

次のときは `feat` を使います。

- これまでなかった bootstrap 機能や運用補助を追加した
- 新しい manifest 群や script を導入した

### `refactor`

次のときは `refactor` を使います。

- 挙動を変えずに file 配置や構造を整理した
- script や manifest の責務分離を行った

### `chore`

次のときは `chore` を使います。

- 小さな保守作業
- version pin 更新
- generated ではないが利用者影響の弱い雑多な整理

`fix` や `refactor` と説明できるなら、そちらを優先します。

## 本文ルール

本文は必要なときだけ付けます。

次の場合は本文を付けます。

- 複数の変更点があり、件名だけでは意図が不足する
- 破壊的変更の移行手順が必要
- 変更理由を残さないと将来の判断が難しい

本文の行長はおおむね 72 文字以内に保ちます。

例:

```text
fix(nix): prefer repo kubeconfig in dev shell

Detect kubeconfig files inside the repository before falling back to
.secret/kubeconfig so local kubectl commands work without extra exports.
```

## Breaking Change

後方互換性を壊す場合は `!` を付け、必要なら footer も付けます。

```text
refactor(bootstrap)!: split Calico bootstrap manifests

BREAKING CHANGE: apply paths changed from manifests/calico to
manifests/bootstrap/calico.
```

## Repo 固有の判断基準

- `README.md` と `docs/` だけなら、通常は `docs`
- `flake.nix` による開発体験や既定値の修正は、通常は `fix(nix)` か `chore(nix)`。挙動修正なら `fix` を優先する
- `scripts/ops/prepare-cluster-access.sh` のような利用手順に直結する script 変更は、通常は `fix(scripts)` か `feat(scripts)`
- `manifests/bootstrap/` の挙動修正は `fix(bootstrap)`、新規導入は `feat(bootstrap)`
- repo rename や構造整理は `refactor(repo)` または `docs` を検討する

## 禁止事項

- 差分を列挙しただけの subject にしない
- `update`, `tweak`, `misc`, `stuff`, `changes` のような曖昧語で終わらせない
- 複数 type を 1 つの件名へ混ぜない
- 実際は挙動変更なのに `docs` へ逃がさない
- secret、token、実 IP、実ホスト名を件名や本文へ書かない

## 出力契約

AI は、最終的に次のどちらかを返します。

- 推奨 commit message を 1 つ
- 候補が割れる場合のみ、第一候補と第二候補を 1 行ずつ

説明が必要なら 1 から 3 文で理由を添えてよいですが、冗長な解説は不要です。

## 実行コマンド

commit message を作る直前に、repo ルートで次を実行します。

```sh
git status --short
git diff --cached --stat
git diff --cached --name-status
git diff --cached --no-color --binary
```

commit 対象は staged diff です。`git diff` のみを実行して unstaged diff を入力にしたり、
`git diff HEAD` で staged diff と unstaged diff を混ぜたりしません。

staged diff が空の場合は commit message を作らず、先に commit 対象を stage するよう
利用者へ伝えます。

## 実行手順

1. `git status --short` で staged、unstaged、untracked の状態を把握する
2. `git diff --cached --stat` と `git diff --cached --name-status` で commit 対象の全体像を確認する
3. `git diff --cached --no-color --binary` を読み、何が変わったかではなく何を達成したかを要約する
4. その要約から type を 1 つ選ぶ
5. 主対象が明確なら scope を付ける
6. 50 から 72 文字程度で subject を組み立てる
7. 曖昧語、末尾の句点、差分列挙表現を除く
8. 必要な場合だけ本文を付ける

## AI への指示文

以下をそのまま commit message 生成エージェントの基本指示として使えます。

```text
You are a Conventional Commit writer for this repository.
Read the diff and return the single best commit message.
Use the format <type>(<scope>): <subject> when scope helps, otherwise <type>: <subject>.
Prefer these types: feat, fix, docs, refactor, ci, test, build, chore, revert.
Choose the type from the user-visible intent of the change, not from the file extension.
Keep the subject in English, imperative, concise, and without a trailing period.
Avoid vague subjects such as update, tweak, misc, or changes.
Use docs for documentation-only changes, fix for broken behavior, feat for net-new behavior, and refactor for non-behavioral restructuring.
Return one message by default. Add a body only when the rationale or migration impact would otherwise be unclear.
Do not include secrets, tokens, real IP addresses, or real hostnames in the commit message.
```

## 非対象

この定義には次を含めません。

- commit hook の実装
- PR title の自動変換
- release note の生成
- semantic versioning の自動判定
