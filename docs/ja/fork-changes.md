<!--
Copyright (c) 2026 MICHISUU Inc.
SPDX-License-Identifier: MIT
-->

# フォーク独自の変更一覧

`michisuuinc/opengrc` は [getprobo/probo](https://github.com/getprobo/probo) のフォークです。この文書は、上流からの変更点をまとめたもので、上流の更新を取り込むたびに確認します。

- **起点**：コミット `4207ca3`（タグ `fork-base-4207ca3`、2026-10-13）
- **方針**：上流のファイルの書き換えは最小限にし、独自の機能は新しいファイルに分けます。上流のファイルを書き換えたときは、必ずこの一覧に追加します。

## 上流のファイルを書き換えたもの

上流の取り込み時に衝突する可能性があります。取り込むたびに、上流で同じ箇所が直っていないかを確認してください。直っていれば、上流の修正を採用し、この一覧から外します。

| ファイル | 変更内容 | 理由 | PR | 上流で直ったか確認する方法 |
| --- | --- | --- | --- | --- |
| `pkg/coredata/evidence.go` | `Evidence.Upsert` の INSERT に `organization_id` を追加 | 内部統制の取り込みで「求める証跡」を登録すると、NOT NULL 制約の違反でエラーになっていた | [#2](https://github.com/michisuuinc/opengrc/pull/2) | 上流の `Evidence.Upsert` に `organization_id` があるか |
| `pkg/probo/internal_control_service.go` | 取り込み時の証跡に `OrganizationID` と `InternalControlID` を設定 | 同上。`internal_control_id` が空で外部キー制約の違反になっていた | [#2](https://github.com/michisuuinc/opengrc/pull/2) | 上流の `Import` で証跡にこの 2 つを設定しているか |
| `e2e/console/internal_control_test.go` | テスト `TestInternalControl_ImportWithRequestedEvidences` を追加 | 上の修正の確認（ファイルの末尾に追加しただけ） | [#2](https://github.com/michisuuinc/opengrc/pull/2) | 上流が修正を取り込んだら、上流のテストに合わせて削除してよい |

## 新しく追加したもの

上流のファイルとは重ならないので、取り込み時に衝突しません。ただし、上流で英語の文言やデータが変わったときは、日本語版の更新が必要です（下の「取り込みの手順」）。

| 種類 | 場所 | 内容 | PR |
| --- | --- | --- | --- |
| 日本語訳 | `apps/console/src/**/_locales/ja-JP.json`（7 ファイル）、`apps/employee-portal/src/**/_locales/ja-JP.json`（6 ファイル） | 画面の翻訳 | [#1](https://github.com/michisuuinc/opengrc/pull/1) |
| フレームワーク | `apps/console/public/data/frameworks/ja-JP/`（ISO 27001・ISO 27701）、`.../frameworks/jp/`（Pマーク） | 日本語版のフレームワーク。取り込み画面からアップロードして使う | [#1](https://github.com/michisuuinc/opengrc/pull/1) |
| 内部統制 | `apps/console/public/data/internal-controls/jp/` | Pマークと ISO 27001 に紐付けた内部統制 39 件 | [#1](https://github.com/michisuuinc/opengrc/pull/1)、[#3](https://github.com/michisuuinc/opengrc/pull/3) |
| 文書 | `docs/ja/` | 訳語表（`glossary.md`）、用語解説（`terms.md`）、この一覧、Pマークの方針・規程・公表文・書式のひな形（`templates/`、18 本） | [#1](https://github.com/michisuuinc/opengrc/pull/1)、[#5](https://github.com/michisuuinc/opengrc/pull/5) ほか |
| チェック | `scripts/i18n/`、`.github/workflows/i18n-check.yaml` | 翻訳ファイル・フレームワーク・内部統制のデータが英語版や取り込み形式と食い違っていないかの検査 | [#1](https://github.com/michisuuinc/opengrc/pull/1)、[#3](https://github.com/michisuuinc/opengrc/pull/3) |
| 社内運用環境 | `deploy/aws-ec2/` | AWS EC2 への構築一式と手順書 | [#4](https://github.com/michisuuinc/opengrc/pull/4) |

## リポジトリの設定（コード以外）

| 設定 | 内容 |
| --- | --- |
| ブランチ保護 | ルールセット `protect-main`：`main` への直接の push・force push・削除を禁止。変更は PR を通す（承認は不要） |
| 無効にしたワークフロー | `make`、`trufflehog`。上流専用の実行環境（runs-on）が必要で、フォークでは動かないため |
| `gh` の既定のリポジトリ | `michisuuinc/opengrc`（PR の作成先を誤って上流にしないため） |
| ローカルの `upstream` | push 先を `DISABLED` にしている（取り込み専用） |

## 上流の取り込みの手順

`main` は保護されているので、取り込みも PR で行います。

```
git fetch upstream
git switch -c sync/upstream-YYYYMMDD upstream/main
git merge origin/main        # フォークの変更を乗せる。衝突したらここで解消する
git push -u origin sync/upstream-YYYYMMDD
gh pr create --repo michisuuinc/opengrc --base main --head sync/upstream-YYYYMMDD --title "Sync with upstream"
```

PR をマージする前に、次を確認します。

- [ ] 「上流のファイルを書き換えたもの」の各ファイルで、衝突がないか。上流で直っていれば、上流の修正を採用し、この一覧を更新したか
- [ ] CI の `i18n-check` が通ったか。失敗したら、上流で追加・変更された英語のキーを日本語に訳す（`node scripts/i18n/check-locales.mjs` で場所がわかる）
- [ ] 上流でフレームワークの英語版（`ISO27001-2022.json` など）が変わっていないか。変わっていれば日本語版も合わせる（`node scripts/i18n/check-frameworks.mjs`）
- [ ] Go のコードが変わっている場合は、手元で E2E テストを実行したか（フォークでは Go の CI が動かないため。`TZ=UTC` を付けて実行する）
- [ ] マージ後、本番に反映する場合は `deploy/aws-ec2/README.md` の「更新」の手順に従う
