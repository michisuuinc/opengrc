// Copyright (c) 2026 MICHISUU Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	templatesDir      = "../../docs/ja/templates"
	exampleConfigPath = "../../docs/ja/templates/config.example.yaml"

	minimalConfig = `
会社:
  会社名: 株式会社テスト
  代表者氏名: 山田 太郎
文書:
  制定日: 2026-11-01
`

	// allChoicesConfig turns on every optional block, so the rendering of
	// each "する／あり" branch is exercised.
	allChoicesConfig = `
会社:
  会社名: 株式会社テスト
  代表者氏名: 山田 太郎
  郵便番号: 100-0001
  所在地: 東京都千代田区千代田 1-1
  事業の内容: ソフトウェアの開発
  問合せ窓口: 個人情報保護窓口
  電話番号: 03-0000-0000
  メールアドレス: privacy@example.com
  公表ページのURL: https://example.com/privacy/
  管理区域: 書庫
  取扱区域: 執務スペース
体制:
  個人情報保護管理者: 佐藤 花子
  個人情報保護管理者の所属・職名: 管理部 部長
  個人情報保護監査責任者: 鈴木 一郎
  代行者: 田中 次郎
  情報システム担当者: 高橋 三郎
文書:
  制定日: 2026-11-01
運用:
  JIPDECへの報告先: 報告先
  JIPDECへの報告期限: 報告期限
  JIPDECの報告の手引き: 手引き
選択:
  第三者提供: する
  第三者提供の内容: 提供の内容
  共同利用: する
  共同利用する項目: 項目
  共同利用者の範囲: 範囲
  共同利用の目的: 目的
  共同利用の責任者の名称: 名称
  共同利用の責任者の住所: 住所
  共同利用の責任者の代表者: 代表者
  外国にある第三者への提供: する
  外国提供の目的: 目的
  提供先の国: 国
  外国の提供先の種類: 種類
  外国でのデータの保管: する
  保管する国: 国
  保管先の事業者の種類: 種類
  Cookieなどの外部送信: あり
  匿名加工情報・仮名加工情報: 扱う
  認定個人情報保護団体: 加入している
  認定個人情報保護団体の名称: 団体
  苦情の申出先の名称: 申出先
  団体の住所: 住所
  団体の電話番号: 電話番号
  手数料: あり
  手数料の金額: 1,000 円
  手数料の支払方法: 振込
  手数料の支払期限: 1 か月
  事務所の仕切り: なし
外部送信:
  - 送信先: 解析事業者
    送信する情報: 閲覧したページ
    目的: 分析
`
)

func mustParseConfig(t *testing.T, yaml string) *Config {
	t.Helper()

	cfg, err := ParseConfig([]byte(yaml))
	require.NoError(t, err)
	return cfg
}

func TestParseConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg := mustParseConfig(t, minimalConfig)

	assert.Equal(t, "株式会社テスト", cfg.Values["会社名"])
	assert.Equal(t, "代表取締役", cfg.Values["代表者の役職"])
	assert.Equal(t, "3年", cfg.Values["記録の保存期間"])
	assert.Equal(t, "PMS", cfg.Values[VariableNumberPrefix])
	assert.Equal(t, "2026年11月1日", cfg.Values[VariableEnactedOn])
	assert.Equal(t, "2026年11月1日", cfg.Values[VariableRevisedOn], "改訂日 defaults to 制定日")
	assert.Equal(t, "しない", cfg.Choices["第三者提供"])
	assert.Equal(t, DefaultPurposes, cfg.Purposes)
}

func TestParseConfig_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		yaml    string
		message string
	}{
		{
			name:    "unknown variable",
			yaml:    "会社:\n  会社の名前: テスト\n",
			message: "定義されていない項目です",
		},
		{
			name:    "variable in the wrong section",
			yaml:    "運用:\n  会社名: テスト\n",
			message: "「会社」に書いてください",
		},
		{
			name:    "invalid choice",
			yaml:    "選択:\n  第三者提供: たまに\n",
			message: "は選べません",
		},
		{
			name:    "invalid date",
			yaml:    "文書:\n  制定日: 2026年11月1日\n",
			message: "日付は 2026-11-01 の形式で",
		},
		{
			name:    "nested value",
			yaml:    "会社:\n  会社名:\n    - テスト\n",
			message: "文字・数字・日付のいずれか",
		},
		{
			name:    "multi-line value",
			yaml:    "会社:\n  会社名: |\n    株式会社\n    テスト\n",
			message: "改行を含めずに",
		},
		{
			name:    "misspelled list field",
			yaml:    "利用目的:\n  - 区分: お客様\n    保有個人デ一タ: false\n",
			message: "not found in type",
		},
		{
			name:    "malformed yaml",
			yaml:    "会社: [\n",
			message: "cannot parse configuration",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				t.Parallel()

				_, err := ParseConfig([]byte(tt.yaml))
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.message)
			},
		)
	}
}

func TestParseConfig_BlankValueUsesDefault(t *testing.T) {
	t.Parallel()

	cfg := mustParseConfig(t, minimalConfig+"運用:\n  記録の保存期間:\n")

	assert.Equal(t, "3年", cfg.Values["記録の保存期間"])
}

func TestRender_EscapesTableCells(t *testing.T) {
	t.Parallel()

	cfg := mustParseConfig(t, minimalConfig+"外部送信:\n  - 送信先: A|B\n    送信する情報: 閲覧履歴\n    目的: 分析\n")

	out, problems := Render("20-test.md", "20", "<!-- list 外部送信 -->", cfg)

	require.Empty(t, problems)
	assert.Equal(t, "| A\\|B | 閲覧履歴 | 分析 |\n", out)
}

func TestRender_Substitution(t *testing.T) {
	t.Parallel()

	cfg := mustParseConfig(t, minimalConfig)
	template := strings.Join(
		[]string{
			"<!--",
			"Copyright (c) 2026 MICHISUU Inc.",
			"-->",
			"",
			"# 【会社名】の規程",
			"",
			"文書番号：【文書番号】",
			"※ 作成者向けの注記",
			"> ※ 引用内の注記",
			"",
			"",
			"記入欄：【記入：委託先名】",
		},
		"\n",
	)

	out, problems := Render("10-test.md", "10", template, cfg)

	require.Empty(t, problems)
	assert.Equal(t, "# 株式会社テストの規程\n\n文書番号：PMS-10\n\n記入欄：＿＿＿＿＿＿（委託先名）\n", out)
}

func TestRender_Conditionals(t *testing.T) {
	t.Parallel()

	template := strings.Join(
		[]string{
			"<!-- if 第三者提供=する -->",
			"提供します。",
			"<!-- if 共同利用=する -->",
			"共同利用します。",
			"<!-- endif -->",
			"<!-- else -->",
			"提供しません。",
			"<!-- endif -->",
		},
		"\n",
	)

	tests := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "default choices",
			yaml: minimalConfig,
			want: "提供しません。\n",
		},
		{
			name: "outer block only",
			yaml: minimalConfig + "選択:\n  第三者提供: する\n",
			want: "提供します。\n",
		},
		{
			name: "nested block",
			yaml: minimalConfig + "選択:\n  第三者提供: する\n  共同利用: する\n",
			want: "提供します。\n共同利用します。\n",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				t.Parallel()

				out, problems := Render("20-test.md", "20", template, mustParseConfig(t, tt.yaml))

				require.Empty(t, problems)
				assert.Equal(t, tt.want, out)
			},
		)
	}
}

func TestRender_RemovedBlockSkipsMissingValues(t *testing.T) {
	t.Parallel()

	template := "<!-- if 手数料=あり -->\n【手数料の金額】\n<!-- endif -->\n本文"

	out, problems := Render("21-test.md", "21", template, mustParseConfig(t, minimalConfig))

	require.Empty(t, problems)
	assert.Equal(t, "本文\n", out)
}

func TestRender_MissingValue(t *testing.T) {
	t.Parallel()

	out, problems := Render("01-test.md", "01", "所在地：【所在地】", mustParseConfig(t, minimalConfig))

	require.Len(t, problems, 1)
	assert.Equal(t, "01-test.md:1: 「所在地」が入力されていません", problems[0].String())
	assert.Contains(t, out, "【所在地】")
}

func TestRender_Lists(t *testing.T) {
	t.Parallel()

	cfg := mustParseConfig(
		t,
		minimalConfig+`
利用目的:
  - 区分: お客様
    個人情報: 氏名
    目的: [契約の履行, ご連絡]
  - 区分: 委託
    個人情報: 契約で定めるもの
    目的: [業務の遂行]
    保有個人データ: false
外部送信:
  - 送信先: 解析事業者
    送信する情報: 閲覧したページ
    目的: 分析
`,
	)

	t.Run(
		"public purposes include every row",
		func(t *testing.T) {
			t.Parallel()

			out, problems := Render("20-test.md", "20", "<!-- list 利用目的:公表 -->", cfg)

			require.Empty(t, problems)
			assert.Equal(t, "| お客様 | 氏名 | 契約の履行、ご連絡 |\n| 委託 | 契約で定めるもの | 業務の遂行 |\n", out)
		},
	)

	t.Run(
		"retained purposes skip entrusted data",
		func(t *testing.T) {
			t.Parallel()

			out, problems := Render("21-test.md", "21", "<!-- list 利用目的:保有 -->", cfg)

			require.Empty(t, problems)
			assert.Equal(t, "| お客様の情報 | 契約の履行、ご連絡 |\n", out)
		},
	)

	t.Run(
		"transmissions",
		func(t *testing.T) {
			t.Parallel()

			out, problems := Render("20-test.md", "20", "<!-- list 外部送信 -->", cfg)

			require.Empty(t, problems)
			assert.Equal(t, "| 解析事業者 | 閲覧したページ | 分析 |\n", out)
		},
	)

	t.Run(
		"transmissions are required when the list is rendered",
		func(t *testing.T) {
			t.Parallel()

			_, problems := Render("20-test.md", "20", "<!-- list 外部送信 -->", mustParseConfig(t, minimalConfig))

			require.Len(t, problems, 1)
			assert.Contains(t, problems[0].Message, "「外部送信」の一覧が入力されていません")
		},
	)
}

func TestRender_RenumbersHeadings(t *testing.T) {
	t.Parallel()

	template := strings.Join(
		[]string{
			"## 1. 目的",
			"<!-- if 共同利用=する -->",
			"## 2. 共同利用",
			"<!-- endif -->",
			"## 3. 問合せ",
			"### 3.1 窓口",
			"## 改訂履歴",
		},
		"\n",
	)

	out, problems := Render("20-test.md", "20", template, mustParseConfig(t, minimalConfig))

	require.Empty(t, problems)
	assert.Equal(t, "## 1. 目的\n## 2. 問合せ\n### 3.1 窓口\n## 改訂履歴\n", out)
}

func TestRender_UnbalancedDirectives(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		template string
		message  string
	}{
		{name: "missing endif", template: "<!-- if 手数料=あり -->\n本文", message: "endif が足りません"},
		{name: "stray else", template: "<!-- else -->", message: "対応する if がありません"},
		{name: "stray endif", template: "<!-- endif -->", message: "対応する if がありません"},
		{name: "if without value", template: "<!-- if 手数料 -->\n<!-- endif -->", message: "「名前=値」"},
		{name: "unknown list", template: "<!-- list 委託先 -->", message: "定義されていない一覧です"},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				t.Parallel()

				_, problems := Render("10-test.md", "10", tt.template, mustParseConfig(t, minimalConfig))

				require.Len(t, problems, 1)
				assert.Contains(t, problems[0].Message, tt.message)
			},
		)
	}
}

func TestCheck_Problems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		template string
		message  string
	}{
		{name: "unknown variable", template: "【会社の名前】", message: "定義されていない変数です: 【会社の名前】"},
		{name: "unknown choice", template: "<!-- if 委託=する -->\n<!-- endif -->", message: "定義されていない選択です"},
		{name: "unknown choice value", template: "<!-- if 手数料=たまに -->\n<!-- endif -->", message: "「手数料」に「たまに」はありません"},
		{name: "if without value", template: "<!-- if 手数料 -->\n<!-- endif -->", message: "「名前=値」"},
		{name: "unknown list", template: "<!-- list 委託先 -->", message: "定義されていない一覧です"},
		{name: "stray else", template: "<!-- else -->", message: "対応する if がありません"},
		{name: "stray endif", template: "<!-- endif -->", message: "対応する if がありません"},
		{name: "missing endif", template: "<!-- if 手数料=あり -->", message: "endif が足りません"},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				t.Parallel()

				problems := Check("10-test.md", tt.template)

				require.Len(t, problems, 1)
				assert.Contains(t, problems[0].Message, tt.message)
			},
		)
	}
}

func TestCheck_AcceptsFillInAndDocumentNumber(t *testing.T) {
	t.Parallel()

	assert.Empty(t, Check("30-test.md", "【文書番号】【記入：氏名】【会社名】"))
}

func TestCheckPaths(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	templates := filepath.Join(repo, "docs", "ja", "templates")
	require.NoError(t, os.MkdirAll(templates, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(repo, ".git"), 0o755))
	outside := t.TempDir()

	tests := []struct {
		name    string
		config  string
		out     string
		message string
	}{
		{name: "both outside", config: filepath.Join(outside, "company.yaml"), out: filepath.Join(outside, "out")},
		{name: "example configuration", config: filepath.Join(templates, ExampleConfigFile), out: filepath.Join(outside, "out")},
		{name: "configuration in the repository", config: filepath.Join(repo, "company.yaml"), out: filepath.Join(outside, "out"), message: "設定ファイルはリポジトリの外に"},
		{name: "configuration next to the example", config: filepath.Join(templates, "company.yaml"), out: filepath.Join(outside, "out"), message: "設定ファイルはリポジトリの外に"},
		{name: "output in the repository", config: filepath.Join(outside, "company.yaml"), out: filepath.Join(repo, "out"), message: "出力先はリポジトリの外に"},
		{name: "output is the repository", config: filepath.Join(outside, "company.yaml"), out: repo, message: "出力先はリポジトリの外に"},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				t.Parallel()

				err := checkPaths(templates, tt.config, tt.out)
				if tt.message == "" {
					assert.NoError(t, err)
					return
				}
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.message)
			},
		)
	}
}

func TestTemplates(t *testing.T) {
	t.Parallel()

	templates, err := loadTemplates(templatesDir)
	require.NoError(t, err)
	require.Len(t, templates, 18)

	t.Run(
		"every template passes the check",
		func(t *testing.T) {
			t.Parallel()

			for _, tmpl := range templates {
				assert.Empty(t, Check(tmpl.file, tmpl.content), tmpl.file)
			}
		},
	)

	example, err := os.ReadFile(exampleConfigPath)
	require.NoError(t, err)

	configs := map[string]string{
		"example configuration": string(example),
		"office with partitions": strings.Replace(
			string(example),
			"事務所の仕切り: なし",
			"事務所の仕切り: あり",
			1,
		),
		"all optional blocks": allChoicesConfig,
	}
	for name, yaml := range configs {
		t.Run(
			name,
			func(t *testing.T) {
				t.Parallel()

				cfg := mustParseConfig(t, yaml)

				for _, tmpl := range templates {
					out, problems := Render(tmpl.file, tmpl.number, tmpl.content, cfg)

					assert.Empty(t, problems, tmpl.file)
					assert.NotContains(t, out, "【", tmpl.file)
					assert.NotContains(t, out, "<!--", tmpl.file)
					assert.NotContains(t, out, "\n※", tmpl.file)
				}
			},
		)
	}
}

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run(
		"check",
		func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, 0, run(templatesDir, "", "", true))
		},
	)

	t.Run(
		"writes every document",
		func(t *testing.T) {
			t.Parallel()

			outDir := t.TempDir()

			require.Equal(t, 0, run(templatesDir, exampleConfigPath, outDir, false))

			files, err := filepath.Glob(filepath.Join(outDir, "*.md"))
			require.NoError(t, err)
			assert.Len(t, files, 18)
		},
	)

	t.Run(
		"missing arguments",
		func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, 2, run(templatesDir, "", "", false))
		},
	)

	t.Run(
		"missing templates",
		func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, 2, run(t.TempDir(), "", "", true))
		},
	)

	t.Run(
		"output inside the repository",
		func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, 2, run(templatesDir, exampleConfigPath, templatesDir, false))
			assert.Equal(t, 2, run(templatesDir, exampleConfigPath, filepath.Join(templatesDir, "out"), false))
			assert.Equal(t, 2, run(templatesDir, exampleConfigPath, "../../out", false))
		},
	)

	t.Run(
		"configuration inside the repository",
		func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, 2, run(templatesDir, filepath.Join(templatesDir, "company.yaml"), t.TempDir(), false))
			assert.Equal(t, 2, run(templatesDir, "../../company.yaml", t.TempDir(), false))
		},
	)

	t.Run(
		"incomplete configuration",
		func(t *testing.T) {
			t.Parallel()

			configPath := filepath.Join(t.TempDir(), "company.yaml")
			require.NoError(t, os.WriteFile(configPath, []byte(minimalConfig), 0o600))

			assert.Equal(t, 1, run(templatesDir, configPath, t.TempDir(), false))
		},
	)
}
