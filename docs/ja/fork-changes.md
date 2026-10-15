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
| `pkg/server/api/console/v1/graphql/processing_activity.graphql` | `ProcessingActivity` に `ledgerDetails` フィールドを 1 つ追加 | 個人情報管理台帳（B1）。GraphQL の規約で `extend type` が Mutation にしか使えないため、型の定義に直接追加した | [#15](https://github.com/michisuuinc/opengrc/pull/15) | 上流で同じ名前のフィールドが追加されていないか |
| `pkg/server/api/console/v1/processing_activity_resolvers.go` | `LedgerDetails` リゾルバ（上の追加で gqlgen が生成する関数） | 同上 | [#15](https://github.com/michisuuinc/opengrc/pull/15) | 同上 |
| `pkg/server/api/console/v1/graphql/risk.graphql` | `Risk` に `privacyDetails`・`privacyProcessingActivities` フィールドを追加 | リスク分析表（B2）。Relay のスキーマ結合で `extend type` が使えないため、型の定義に直接追加した | [#16](https://github.com/michisuuinc/opengrc/pull/16) | 上流で同じ名前のフィールドが追加されていないか |
| `pkg/server/api/console/v1/risk_resolvers.go` | `PrivacyDetails`・`PrivacyProcessingActivities` リゾルバ（上の追加で gqlgen が生成する関数） | 同上 | [#16](https://github.com/michisuuinc/opengrc/pull/16) | 同上 |
| `pkg/server/api/console/v1/graphql/audit.graphql` | `Audit` に `privacyDetails`・`privacyAuditors`・`privacyIndependenceConflicts`、`Finding` に `privacyDetails` フィールドを追加 | 内部監査報告書（B6）。Relay のスキーマ結合で `extend type` が使えないため、型の定義に直接追加した | 未定 | 上流で同じ名前のフィールドが追加されていないか |
| `pkg/server/api/console/v1/audit_resolvers.go` | 上の 4 つのリゾルバ（gqlgen が生成する関数。監査員の権限確認は `audit_privacy_helpers.go`） | 同上 | 未定 | 同上 |
| `apps/console/src/hooks/graph/AuditGraph.ts`、`.../audits/AuditDetailsPage.tsx`、`.../findings/FindingDetailsPage.tsx` | 監査の詳細に内部監査報告書のカード、指摘事項の詳細に分類と根拠のカードを差し込む（数行ずつ）。監査のカードはメンバーを一覧できる人にだけ表示し、データは自分のクエリで読む | 同上。監査員の氏名にはメンバー一覧の権限が要るので、権限のない人の監査の画面が壊れないようにする | 未定 | 衝突したら差し込み位置を合わせ直す |
| `pkg/probo/service.go` | `ProcessingActivityLedger`・`RiskPrivacy` サービスの登録（2 行ずつ） | 台帳（B1）とリスク分析表（B2） | [#15](https://github.com/michisuuinc/opengrc/pull/15)、[#16](https://github.com/michisuuinc/opengrc/pull/16) | 衝突したら行を入れ直す |
| `apps/console/src/pages/organizations/risks/RisksPage.tsx`、`.../risks/overview/RiskOverviewPage.tsx` | 「リスク分析表を出力（Excel）」ボタンとリスク分析表の項目のカードを差し込む（数行ずつ） | リスク分析表（B2） | [#16](https://github.com/michisuuinc/opengrc/pull/16) | 衝突したら差し込み位置を合わせ直す |
| `apps/console/src/hooks/graph/ProcessingActivityGraph.ts`、`.../processingActivities/ProcessingActivityDetailsPage.tsx`、`.../ProcessingActivitiesPage.tsx` | 台帳の項目のカードと「台帳を出力（Excel）」ボタンを差し込む（数行ずつ） | 同上 | [#15](https://github.com/michisuuinc/opengrc/pull/15) | 衝突したら差し込み位置を合わせ直す |
| `apps/console/src/_locales/en-US.json` | キー `processingActivityLedger`・`riskPrivacy`・`training`・`nav.trainings`・`auditPrivacy`・`findingPrivacy` を追加 | 台帳（B1）、リスク分析表（B2）、教育（B7）、内部監査報告書（B6）（fr-FR・nl-NL は英語にフォールバック） | [#15](https://github.com/michisuuinc/opengrc/pull/15)、[#16](https://github.com/michisuuinc/opengrc/pull/16)、[#18](https://github.com/michisuuinc/opengrc/pull/18) | 衝突したらブロックを入れ直す |
| `packages/ui/src/Molecules/FrameworkSelector/FrameworkSelector.tsx` | 「新しいフレームワーク」の一覧に Pマーク（`PMARK-JISQ15001-2023`）を 1 件追加 | フレームワークの言語切り替え（A3）。ファイルを手元に用意せずに一覧から取り込めるようにする | [#17](https://github.com/michisuuinc/opengrc/pull/17) | 衝突したら一覧の末尾に入れ直す |
| `apps/console/src/pages/organizations/frameworks/FrameworksPage.tsx`、`.../risks/_components/FormRiskDialog.tsx` | 読み込むファイルのパスを `#/lib/i18n/localizedData` の関数で決める（数行ずつ） | 同上。画面の言語に合わせて日本語版のフレームワーク・リスクテンプレートを読み込む | [#17](https://github.com/michisuuinc/opengrc/pull/17) | 上流で言語ごとのデータの仕組みができたら、そちらに合わせる |
| `pkg/docgen/generator.go`、`pkg/docgen/template.html`、`pkg/docgen/signature_page_template.html` | `DocumentData`・`SignaturePageData` に `Language` を追加し、`<html lang>` に出力。空なら本文から判定（かながあれば `ja`） | PDF の日本語フォント（A1）。`lang="en"` のままだと、日本語の漢字が中国語の字形で表示されることがある | [#17](https://github.com/michisuuinc/opengrc/pull/17) | 上流で文書に言語の項目ができたら、そちらを使う |
| `pkg/probo/document_service.go` | 署名ページの出力で `Language` を渡す（1 行） | 同上。署名者の名前は漢字だけのことが多く、名前からは判定できないため、文書本体の判定結果を使う | [#17](https://github.com/michisuuinc/opengrc/pull/17) | 同上 |
| `pkg/coredata/entity_type_reg.go`、`pkg/coredata/audit_log_resource_type.go` | 教育の 2 つのエンティティ種別（`TrainingPlanEntityType = 1000`、`TrainingParticipantEntityType = 1001`）と、その名前を追加 | 教育（B7）。上流は 137 まで連番で増やしているので、衝突しないようにフォーク独自の番号は 1000 から使う | [#18](https://github.com/michisuuinc/opengrc/pull/18) | 上流の番号が 1000 に近づいたら見直す |
| `pkg/probo/actions.go`、`pkg/probo/policies.go` | アクション `core:training-plan:*` を追加し、閲覧者・監査人に get・list を許可 | 同上 | [#18](https://github.com/michisuuinc/opengrc/pull/18) | 衝突したら行を入れ直す |
| `pkg/probo/service.go` | `Trainings` サービスの登録（2 行）。B6 の `AuditPrivacy` も同じ場所に 2 行 | 同上 | [#18](https://github.com/michisuuinc/opengrc/pull/18) | 同上 |
| `pkg/server/api/console/v1/graphql/organization.graphql`、`organization_resolvers.go` | `Organization.trainingPlans` フィールドとリゾルバ（処理は `training_helpers.go` に置き、1 行で呼ぶ） | 同上。Relay のスキーマ結合で `extend type` が使えないため | [#18](https://github.com/michisuuinc/opengrc/pull/18) | 上流で同じ名前のフィールドが追加されていないか |
| `pkg/server/api/console/v1/base_resolvers.go` | `node(id:)` で教育と受講者を解決する case を追加 | 同上 | [#18](https://github.com/michisuuinc/opengrc/pull/18) | 衝突したら case を入れ直す |
| `apps/console/src/routes.tsx`、`.../shell/GovernanceNavPanel.tsx` | ルート `governance/trainings` と、メニュー「教育」を追加（数行ずつ） | 同上 | [#18](https://github.com/michisuuinc/opengrc/pull/18) | 衝突したら差し込み位置を合わせ直す |
| `go.mod`、`go.sum` | `github.com/xuri/excelize/v2` を追加 | Excel 出力（B0）。BSD-3-Clause。依存の mscfb・msoleps は Apache-2.0、go-deepcopy は MIT | [#15](https://github.com/michisuuinc/opengrc/pull/15) | 上流が同じライブラリを入れたら上流の版に合わせる |

## 新しく追加したもの

上流のファイルとは重ならないので、取り込み時に衝突しません。ただし、上流で英語の文言やデータが変わったときは、日本語版の更新が必要です（下の「取り込みの手順」）。

| 種類 | 場所 | 内容 | PR |
| --- | --- | --- | --- |
| 日本語訳 | `apps/console/src/**/_locales/ja-JP.json`（7 ファイル）、`apps/employee-portal/src/**/_locales/ja-JP.json`（6 ファイル） | 画面の翻訳 | [#1](https://github.com/michisuuinc/opengrc/pull/1) |
| 教育 | マイグレーション `20261015T090000Z.sql`（テーブル `training_plans`・`training_plan_materials`・`training_participants`）、`pkg/coredata/training_*.go`、`pkg/probo/training_{service,progress,sheet}.go`、`.../graphql/training.graphql`・`training_resolvers.go`・`training_helpers.go`・`types/training.go`、`apps/console/src/pages/organizations/trainings/`、`apps/console/src/routes/trainingRoutes.ts`、`e2e/console/training_test.go`、マニュアル 3.3 | Pマークの教育（B7）。年度ごとの教育の計画、教材（文書）、受講者を管理し、実施期間内に教材すべてに署名したか、手入力の受講日で受講を判定する。理解度テストの点数・合否、未受講者の絞り込み、教育実施記録の Excel 出力。記録のある受講者は外せず、その教育は削除できない（記録の保存）。署名の依頼は既存の文書の署名依頼を使う。MCP・CLI・n8n、OAuth2 のスコープには未対応（OAuth2 のクライアントからは使えない。B1 と同じく上流に出すときに追加する） | [#18](https://github.com/michisuuinc/opengrc/pull/18) |
| フレームワーク | `apps/console/public/data/frameworks/ja-JP/`（ISO 27001・ISO 27701）、`.../frameworks/jp/`（Pマーク）、`apps/console/src/lib/i18n/localizedData.ts`、`packages/ui/src/Atoms/Frameworks/JISQ15001.tsx` | 日本語版のフレームワーク。画面が日本語なら「新しいフレームワーク」の一覧から日本語版が取り込まれる。Pマークは一覧に追加（アイコンは JIPDEC の登録商標の PrivacyMark のロゴを使わず、文字だけにした）。日本語版を追加したら `localizedData.ts` の一覧にも追加する | [#1](https://github.com/michisuuinc/opengrc/pull/1)、[#17](https://github.com/michisuuinc/opengrc/pull/17) |
| リスクテンプレート | `apps/console/public/data/risks/ja-JP/`（`risks.json`、`LICENSE.md`） | 上流のリスクテンプレート 82 件の日本語訳。元データと同じ CC BY-SA 4.0 で、クレジットと変更点を `LICENSE.md` に記載。上流で `risks.json` が変わったら訳も直す（件数が変わると検査で失敗する） | [#17](https://github.com/michisuuinc/opengrc/pull/17) |
| ローカルの開発環境 | `compose.override.yaml`、`compose/chrome-ja/Dockerfile` | PDF 出力用の Chrome に日本語フォント（`fonts-noto-cjk`）を入れたイメージ（ローカルの開発用）。`docker compose` をそのまま実行すると自動で読み込まれる。`make stack-up` などは `-f compose.yaml` を指定するので、`DOCKER_COMPOSE_FLAGS="-f compose.override.yaml"` を付ける。上流の `compose.yaml` と CI は変えていない（CI は `COMPOSE_FILE` を指定するので読み込まれない） | [#17](https://github.com/michisuuinc/opengrc/pull/17) |
| 内部統制 | `apps/console/public/data/internal-controls/jp/` | Pマークと ISO 27001 に紐付けた内部統制 39 件 | [#1](https://github.com/michisuuinc/opengrc/pull/1)、[#3](https://github.com/michisuuinc/opengrc/pull/3) |
| 文書 | `docs/ja/` | 訳語表（`glossary.md`）、用語解説（`terms.md`）、この一覧、Pマークの方針・規程・公表文・書式のひな形（`templates/`、18 本）、利用マニュアル（`manual/`） | [#1](https://github.com/michisuuinc/opengrc/pull/1)、[#5](https://github.com/michisuuinc/opengrc/pull/5) ほか |
| チェック | `scripts/i18n/`、`.github/workflows/i18n-check.yaml` | 翻訳ファイル・フレームワーク・内部統制・リスクテンプレートのデータが英語版や取り込み形式と食い違っていないかの検査 | [#1](https://github.com/michisuuinc/opengrc/pull/1)、[#3](https://github.com/michisuuinc/opengrc/pull/3)、[#17](https://github.com/michisuuinc/opengrc/pull/17) |
| 生成ツール | `cmd/ja-docgen/`、`scripts/ja-docs/`、`docs/ja/templates/config.example.yaml`、`docs/ja/templates/variables.md`、`.github/workflows/ja-templates-check.yaml` | 文書ひな形に会社の値を入れて完成版を作るツール、完成版を OpenGRC に登録して統制・内部統制に紐付けるスクリプト、ひな形の検査 | [#10](https://github.com/michisuuinc/opengrc/pull/10)〜[#14](https://github.com/michisuuinc/opengrc/pull/14) |
| 個人情報管理台帳 | `pkg/xlsxexport/`、`pkg/coredata/processing_activity_ledger_details.go`、マイグレーション `20261015T025623Z.sql`（テーブル `processing_activity_ledger_details`）、`pkg/probo/processing_activity_ledger_*.go`、`.../graphql/processing_activity_ledger.graphql` とリゾルバ、`.../processingActivities/_components/ProcessingActivityLedgerCard.tsx`・`_lib/`、`e2e/console/processing_activity_ledger_test.go`、マニュアル第 3 章 | Excel 出力の共通部分（B0）と、Pマークの台帳の追加項目（担当部門・件数・取得の方法・保管の場所と媒体・アクセスできる人の範囲・廃棄の方法）・台帳の Excel 出力（B1）。今は GraphQL と画面だけで、MCP・CLI・n8n には未対応（上流の規約 `contrib/claude/api-surface.md` との差。上流に出すときに追加する） | [#15](https://github.com/michisuuinc/opengrc/pull/15) |
| リスク分析表 | マイグレーション `20261015T033524Z.sql`（テーブル `risk_privacy_details`・`risk_processing_activities`）、`pkg/coredata/personal_data_lifecycle_stage.go`・`risk_privacy_details.go`・`risk_processing_activity.go`・`internal_control_risk_names.go`、`pkg/probo/risk_privacy_*.go`、`.../graphql/risk_privacy.graphql`・`risk_privacy_resolvers.go`・`risk_privacy_helpers.go`・`types/risk_privacy.go`、`.../risks/_components/RiskPrivacyCard.tsx`・`_lib/`、`e2e/console/risk_privacy_test.go`、マニュアル 3.2 | Pマークのリスク分析表（B2）。リスクに個人情報の段階（取得・入力〜消去・廃棄）と、対象の台帳（処理活動）との紐付けを追加し、台帳の業務 × リスクの一覧を Excel で出力する。MCP・CLI・n8n には未対応（B1 と同じ） | [#16](https://github.com/michisuuinc/opengrc/pull/16) |
| 内部監査報告書 | マイグレーション `20261015T120000Z.sql`（テーブル `audit_privacy_details`・`audit_privacy_auditors`・`finding_privacy_details`）、`pkg/coredata/audit_privacy_*.go`・`finding_privacy_*.go`、`pkg/probo/audit_privacy_*.go`、`pkg/xlsxexport` の `BuildWorkbook`（複数シート）、`.../graphql/audit_privacy.graphql`・`audit_privacy_resolvers.go`・`audit_privacy_helpers.go`・`types/audit_privacy.go`、`.../audits/_components/AuditPrivacyReportCard.tsx`・`_lib/`、`.../findings/_components/FindingPrivacyCard.tsx`・`_lib/`、`e2e/console/audit_privacy_test.go`、マニュアル 3.4 | Pマークの内部監査報告書（B6）。監査に範囲・基準・監査員（社内はメンバー、外部は氏名）・被監査部門・総合的な所見、指摘事項に規程の分類（不適合・観察事項・改善の機会。選ばなければ種別から決まる）と根拠を追加し、監査概要と指摘事項一覧の 2 シートの Excel を出力する。監査員が被監査部門の所属なら警告する。新しいアクションは作らず、上流の監査・指摘事項の権限を使う。MCP・CLI・n8n には未対応（B1 と同じ） | 未定 |

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
- [ ] 上流でリスクテンプレート（`apps/console/public/data/risks/risks.json`）が変わっていないか。変わっていれば `ja-JP/risks.json` も合わせる（同じ検査で件数の違いを検出する）
- [ ] 上流の `compose.yaml` で Chrome のイメージ（digest）が変わっていないか。変わっていれば `compose/chrome-ja/Dockerfile` の `FROM` も合わせる
- [ ] Go のコードが変わっている場合は、手元で E2E テストを実行したか（フォークでは Go の CI が動かないため。`TZ=UTC` を付けて実行する）
