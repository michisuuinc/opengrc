<!--
Copyright (c) 2026 MICHISUU Inc.
SPDX-License-Identifier: MIT
-->

# 社内運用環境（AWS EC2 1 台）

`https://grc.michisuu.com` で動かす社内運用環境の構成と手順です。AWS の東京リージョン（ap-northeast-1）に EC2 を 1 台置き、Docker Compose で動かします。

## 構成

```
インターネット ── TCP 443 / 80 のみ ──▶ EC2（t3.large、Ubuntu 24.04、SSH なし）
                                        ├ caddy    HTTPS 終端（Let's Encrypt）→ probo:8080
                                        ├ probo    このフォークから作ったイメージ（ECR、ダイジェスト固定）
                                        ├ postgres 外部に出ないネットワーク（db）
                                        └ chrome   PDF 生成（render ネットワーク）
S3：ファイル（michisuu-opengrc-files）、DB バックアップ（michisuu-opengrc-backup）
SSM Parameter Store：秘密情報（/opengrc/*）
SES：メール送信（no-reply@michisuu.com）
CloudWatch Logs：コンテナのログ、Session Manager の操作記録（400 日保存）
```

| ファイル | 役割 | 実行する場所 |
| --- | --- | --- |
| `provision.sh` | AWS 資源の作成（段階ごと） | 手元の PC |
| `build-image.sh` | イメージのビルドと ECR への push | 手元の PC |
| `deploy.sh` | 本番への反映（SSM Run Command 経由） | 手元の PC |
| `user-data.sh` | EC2 の初回起動時の設定 | EC2（自動） |
| `fetch-secrets.sh` | 実行時の設定 `runtime.conf` の生成 | EC2（deploy.sh から） |
| `imds-guard.sh` | probo 以外のコンテナからインスタンスメタデータへの通信を遮断 | EC2（systemd） |
| `backup.sh` | DB のバックアップ（毎日 02:30） | EC2（systemd タイマー） |
| `compose.yaml`・`Caddyfile`・`probod-settings.conf`・`postgres-init.sh` | サービスの定義と設定 | EC2 |

## デプロイ前のセキュリティ確認

構築を始める前に、次の項目を確認してください。セキュリティ専門のレビュー（2026-10-14 実施）の指摘を反映済みです。

### この構成で対策済みのもの

| 項目 | 対策 |
| --- | --- |
| SSH | 使わない。キーペアを付けない、セキュリティグループで 22 番を開けない、`openssh-server` と `ec2-instance-connect` を削除し、ssh を mask する。`provision.sh infra` を実行するたびに、セキュリティグループに TCP 80/443 以外の規則がないかを確認し、あれば止まる |
| サーバーへの接続 | SSM Session Manager / Run Command だけ。すべての操作を CloudWatch Logs に記録する（`provision.sh audit`） |
| 公開ポート | caddy の 80/443 だけ。Postgres・Chrome・probod の管理用ポートは外に出ない |
| コンテナ間の分離 | ネットワークを役割ごとに分ける（edge・db・render）。DB のネットワークはインターネットにも出られない |
| 権限情報の取得 | インスタンスメタデータ（IMDSv2 必須）に届くのは probo コンテナだけ（`imds-guard.sh`）。Chrome などが乗っ取られても、AWS の権限情報は取れない |
| IAM（EC2） | ファイル用バケットの読み書き、バックアップ用バケットへの書き込みだけ（読み取り・削除は不可）、対象の ECR リポジトリの取得、`/opengrc/*` の秘密の読み取りだけ。秘密の書き換え・削除は明示的に拒否 |
| IAM（メール送信） | SES 専用のユーザー。検証済みドメインから `no-reply@michisuu.com` として送る権限だけ |
| 秘密情報 | 手元で生成して SSM（SecureString）に保存。画面にもコマンドの引数にも出さない。サーバーのディスクには、DB のパスワードを root だけが読めるファイルに置くだけ |
| 本番で動くコード | `deploy.sh` は、`origin/main` に取り込み済みのコミットだけを、SHA を指定して反映する。イメージは ECR のダイジェスト（`@sha256:…`）で固定し、タグの上書きは禁止 |
| データの保護 | S3 は非公開・暗号化・TLS 必須・バージョン管理（古い版は 30 日で削除）。EBS は暗号化し、毎日スナップショット（7 世代）。DB のダンプは毎日 S3 へ（30 日保存） |
| 通信 | HTTPS のみ（HSTS）。セキュリティヘッダーを付与。メールは STARTTLS 必須 |
| 監査 | CloudTrail（なければ作成、ログファイルの検証あり）、GuardDuty（なければ有効化）、コンテナとセッションのログは 400 日保存 |
| ログイン | パスワードでの新規登録は無効。普段は Google ログイン（Workspace 側で MFA 必須）。Google が使えないときのために、管理者 3 人だけが非常用のパスワードを持つ（下の「非常用のパスワード」） |

### Minami さんに確認・実施してほしいもの

これらは AWS アカウントや GitHub 全体の設定なので、スクリプトでは変更していません。

- [ ] **SSM の接続を許す人を絞る**：`ssm:StartSession` と `ssm:SendCommand` は、サーバーの管理者権限と同じです。許可は管理者のロールやユーザーだけにし、MFA を必須にしてください。対象の EC2 には `Project=opengrc` のタグを付けています。IAM ポリシーの条件 `ssm:resourceTag/Project` で、この EC2 に限定できます
- [ ] **AWS アカウントのルートユーザー**：MFA が設定されていて、普段は使っていないこと
- [ ] **GitHub の `main` を保護する**：`michisuuinc/opengrc` の `main` に、レビュー必須・force push 禁止のブランチ保護を設定する。`main` に入ったコードがそのまま本番で動くためです
- [ ] **Google の OAuth 同意画面を「内部」にする**：Google ログインは、ほかの会社の Workspace アカウントも受け付けてしまいます（アプリ側でドメインを制限していません）。同意画面を「内部」にすると、michisuu.com のアカウントだけに限定できます
- [ ] **暗号化キーの保管**：`/opengrc/encryption-key` を、AWS の外（パスワード管理ツールなど）にも保管する。SSM から消えると、暗号化されたデータを復元できません
- [ ] **AI 機能を使わないことを社内に周知する**：AI 機能はダミーのキーで無効化しています。誤って使うと、データが外部（OpenAI）に送られて失敗します
- [ ] **アクセスキーの棚卸し**：手元の PC で使う AWS の認証情報が、長期のアクセスキーなら、できれば IAM Identity Center などの一時的な認証に切り替える

### 残っているリスク（受け入れるかどうかの判断が必要）

| リスク | 内容 | 必要なら取れる対策 |
| --- | --- | --- |
| バックアップが同じアカウント・同じリージョンにある | アカウントそのものを乗っ取られたり、リージョン全体が止まったりすると失う | 別アカウントや大阪リージョンへのコピー |
| DB のパスワードがコンテナの環境変数にある | サーバーの root・docker グループの人は `docker inspect` で見られる | Docker secrets・`*_FILE` 形式への移行 |
| サーバーからの外向き通信は制限していない | 乗っ取られた場合に外部へ送信できる | 外向き通信の許可リスト、VPC エンドポイント |
| 非常用のパスワードでのログインに MFA がない | パスワードが漏れると、MFA なしでログインされる | 3 人だけに限定し、長く使い回しのないパスワードにする。Phase 2 で MFA（D1〜D3）を追加する |
| EC2 を削除しても EBS は残る（`DeleteOnTermination=false`） | 廃止時に個人情報を含むボリュームが残る | 廃止手順でボリュームとスナップショットを削除する |

## 構築手順

**前提**：手元の PC に次がそろっていること。
- AWS CLI（このアカウントの管理者権限）、Docker（buildx）
- Go 1.27・Node 24（`go.mod`・`.nvmrc` の版）
- このリポジトリの作業ディレクトリ（`deploy/aws-ec2` が `main` に入っていること）

すべて `deploy/aws-ec2` で実行します。

1. **秘密情報を作る**（無料）
   ```
   ./provision.sh secrets
   ```
   終わったら、`/opengrc/encryption-key` を AWS の外にも保管してください（上の確認項目）。
2. **メール送信用の認証情報を作る**
   ```
   ./provision.sh ses-smtp
   ```
3. **ECR を作り、イメージをビルドして push する**
   ```
   ./provision.sh ecr
   ./build-image.sh
   ```
   作業ディレクトリに未コミットの変更があると止まります。
4. **監査の設定**（確認あり。アカウント全体の設定を変えます）
   ```
   ./provision.sh audit
   ```
5. **EC2 などを作る**（確認あり。月 90〜100 ドル程度）
   ```
   ./provision.sh infra
   ```
   最後に表示される Elastic IP を控えます。EC2 は `origin/main` のコミットで初期設定されます。
6. **DNS を登録する**（お名前.com）
   - `grc.michisuu.com` の A レコード → 手順 5 の Elastic IP
   - SES の DKIM 用 CNAME 3 件 → 次のコマンドで表示されます
   ```
   ./provision.sh ses-domain
   ```
   数十分〜数時間で反映されます。もう一度実行すると、検証の状態を確認できます。
7. **起動する**
   ```
   ./deploy.sh
   ```
   EC2 の初期設定（数分）が終わっていないと止まります。そのときは少し待ってから再実行してください。
8. **Google ログイン**
   1. Google Cloud コンソールで OAuth クライアントを作ります。
      - 同意画面は「内部」、種類は「ウェブアプリケーション」
      - 承認済みのリダイレクト URI：`https://grc.michisuu.com/api/connect/v1/oidc/google/callback`
      - 承認済みの JavaScript 生成元：`https://grc.michisuu.com`
   2. クライアント ID と秘密を保存します。
      ```
      ./provision.sh google
      ```
   3. `probod-settings.conf` の `PROBOD_AUTH_GOOGLE_*` の 2 行のコメントを外し、PR でマージしてから `./deploy.sh` を実行します。

9. **非常用のパスワードを設定する**（管理者 3 人）

   アプリにはパスワードでログインするときの MFA がありません（Phase 2 の D1〜D3 で追加予定）。そのため、パスワードは Google が使えないときの非常用として、SSM で接続できる管理者 3 人だけが持ちます。

   1. 3 人とも、まず Google でログインしてアカウントを作ります。
   2. 一度ログアウトし、ログイン画面の「パスワードを忘れた」から、自分のメールアドレス（@michisuu.com）にパスワード設定のメールを送ります。メールのリンクからパスワードを設定します。
   3. パスワードは長く（20 文字以上）、ほかで使っていないものにし、パスワード管理ツールに保管します。
   4. 3 人以外のメンバーはパスワードを設定しないことを、社内のルールにします。

   **Google が動いているうちに設定してください。** Google が使えないときは、@michisuu.com のメール（Gmail）も受け取れない可能性が高く、その時点ではパスワードを設定できません。新規登録は無効のままでも、既存のアカウントはこの方法でパスワードを設定できます（新規登録の無効化で止まるのは、パスワードでの新規登録だけです）。

## 動作確認

- [ ] `https://grc.michisuu.com` が有効な証明書で開き、画面が日本語で表示される
- [ ] Google（michisuu.com）のアカウントでログインでき、組織を作成できる
- [ ] 管理者 3 人が、非常用のパスワードでもログインできる（構築手順 9）
- [ ] 招待メールが @michisuu.com に届き、DKIM が「pass」になっている（Gmail の「メッセージのソースを表示」）
- [ ] Pマーク・ISO 27001 のフレームワークと内部統制を取り込める
- [ ] 証跡のファイルをアップロード・ダウンロードできる
- [ ] 文書の PDF を出力でき、日本語が文字化けしない（化ける場合は Phase 2 の A1 で対応）
- [ ] 外部から 80・443 以外のポートが閉じている（例：`nc -vz <IP> 22`、`5432`、`8080`、`9222` がすべて失敗する）
- [ ] CloudWatch Logs の `/opengrc/containers` にログが出ていて、エラーがない
- [ ] バックアップを手動で実行し、S3 にダンプができる（下の「バックアップ」）
- [ ] 復元の練習を 1 回行う（下の「復元」）

## 運用

### 更新
1. 変更を PR で `main` にマージする
2. イメージが変わる場合は `./build-image.sh` を実行する
3. `./deploy.sh` を実行する（特定のコミットなら `./deploy.sh <SHA>`。`main` に入っているものに限る）

### サーバーに入る
```
aws ssm start-session --target <インスタンス ID>
```
操作はすべて CloudWatch Logs の `/opengrc/ssm-sessions` に記録されます。

### バックアップ
- **DB**：毎日 02:30 に `s3://michisuu-opengrc-backup/db/` へ保存されます（30 日保存）。手動で実行する場合は、サーバーで `sudo systemctl start opengrc-backup.service` を実行し、`journalctl -u opengrc-backup` で結果を確認します。
- **EBS**：毎日 03:00 にスナップショットを取得します（7 世代）。
- **ファイル**：S3 のバージョン管理で、削除・上書きから 30 日間は戻せます。

### 復元
EC2 のロールはバックアップを読めないので、復元は管理者の権限で行います。
1. 手元の PC でダンプを取得します（`aws s3 cp s3://michisuu-opengrc-backup/db/<ファイル> .`）。
2. サーバーに送ります。Session Manager でつないだうえで、一時的な S3 の署名付き URL を使うなどの方法があります。
3. 復元します：`docker compose --env-file runtime.conf exec -T postgres pg_restore -U postgres -d probod --clean --if-exists < <ファイル>`
4. 練習するときは、本番の DB ではなく、別名の DB（例：`probod_restore_test`）に復元して件数を確かめます。

### SMTP 認証情報の更新
1. IAM の `opengrc-ses-smtp` に新しいアクセスキーを作る前に、SSM の `/opengrc/smtp-user` と `/opengrc/smtp-password` を削除します。
2. 古いアクセスキーを削除し、`./provision.sh ses-smtp` を実行します。
3. `./deploy.sh` を実行します。

### 廃止するとき
EC2 を削除しても EBS ボリュームは残ります。ボリューム、スナップショット、S3 のバケット（ファイル・バックアップ）を、社内の廃棄手順に沿って削除してください。

## 補足
- **コンプライアンスポータル**：社内運用では公開しないので、外部には出していません（`PROBOD_TRUST_CENTER_TLS_MODE=external`、ポート 8090 は内部のみ）。
- **メールの送信先**：SES は「サンドボックス」のままでも、検証済みドメイン（michisuu.com）宛には送れます。社外に送る必要が出たら、SES の本番アクセスを申請します。
- **固定しているイメージの更新**：caddy・postgres・chrome のイメージはダイジェストで固定しています。セキュリティ更新のため、3 か月に 1 回程度、新しい版のダイジェストに更新してください。
