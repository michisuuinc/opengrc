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

type (
	// Variable is a value substituted for 【name】 in the templates. An empty
	// Default means the value must come from the configuration file whenever
	// the variable appears in the rendered text.
	Variable struct {
		Section string
		Name    string
		Default string
	}

	// Choice selects which conditional blocks of the templates are kept.
	Choice struct {
		Name    string
		Values  []string
		Default string
	}

	// PurposeRow is one row of the purposes of use (利用目的).
	PurposeRow struct {
		Category string   `yaml:"区分"`
		Items    string   `yaml:"個人情報"`
		Purposes []string `yaml:"目的"`
		Retained *bool    `yaml:"保有個人データ"`
	}

	// TransmissionRow is one external transmission (Cookie などの外部送信).
	TransmissionRow struct {
		Recipient   string `yaml:"送信先"`
		Information string `yaml:"送信する情報"`
		Purpose     string `yaml:"目的"`
	}
)

const (
	SectionCompany      = "会社"
	SectionOrganization = "体制"
	SectionDocument     = "文書"
	SectionOperation    = "運用"
	SectionChoice       = "選択"

	VariableDocumentNumber = "文書番号"
	VariableEnactedOn      = "制定日"
	VariableRevisedOn      = "改訂日"
	VariableNumberPrefix   = "文書番号の接頭辞"

	ListPurposesPublic    = "利用目的:公表"
	ListPurposesRetained  = "利用目的:保有"
	ListTransmissions     = "外部送信"
	FillInPrefix          = "記入："
	DocumentNumberDefault = "PMS"
)

var (
	// Variables lists every 【name】 the templates may use, outside 記入欄.
	Variables = []Variable{
		// 会社
		{Section: SectionCompany, Name: "会社名"},
		{Section: SectionCompany, Name: "代表者の役職", Default: "代表取締役"},
		{Section: SectionCompany, Name: "代表者氏名"},
		{Section: SectionCompany, Name: "郵便番号"},
		{Section: SectionCompany, Name: "所在地"},
		{Section: SectionCompany, Name: "事業の内容"},
		{Section: SectionCompany, Name: "適用範囲", Default: "当社の全事業において、全従業者が取り扱うすべての個人情報"},
		{Section: SectionCompany, Name: "問合せ窓口"},
		{Section: SectionCompany, Name: "電話番号"},
		{Section: SectionCompany, Name: "メールアドレス"},
		{Section: SectionCompany, Name: "公表ページのURL"},
		{Section: SectionCompany, Name: "受付時間", Default: "平日 10:00〜17:00、土日祝日と年末年始を除く"},
		{Section: SectionCompany, Name: "管理区域"},
		{Section: SectionCompany, Name: "取扱区域"},

		// 体制
		{Section: SectionOrganization, Name: "個人情報保護管理者"},
		{Section: SectionOrganization, Name: "個人情報保護管理者の所属・職名"},
		{Section: SectionOrganization, Name: "個人情報保護監査責任者"},
		{Section: SectionOrganization, Name: "代行者"},
		{Section: SectionOrganization, Name: "部門責任者", Default: "各部門の長"},
		{Section: SectionOrganization, Name: "情報システム担当者"},
		{Section: SectionOrganization, Name: "監査員", Default: "個人情報保護監査責任者が指名する従業者（自部門は監査しない）"},
		{Section: SectionOrganization, Name: "管理者権限を持つ者", Default: "情報システム担当者と、業務上必要と個人情報保護管理者が認めた者"},
		{Section: SectionOrganization, Name: "就業規則の該当条項", Default: "就業規則の懲戒に関する規定"},
		{Section: SectionOrganization, Name: "外部の相談先", Default: "顧問弁護士"},
		{Section: SectionOrganization, Name: "外部の専門家の連絡先", Default: "個人情報保護管理者が別に定める連絡先一覧"},
		{Section: SectionOrganization, Name: "緊急時の連絡手段", Default: "電話またはチャット"},
		{Section: SectionOrganization, Name: "経理の手続き", Default: "社内の経理の規程"},

		// 文書
		{Section: SectionDocument, Name: VariableEnactedOn},
		{Section: SectionDocument, Name: VariableRevisedOn},
		{Section: SectionDocument, Name: VariableNumberPrefix, Default: DocumentNumberDefault},

		// 運用
		{Section: SectionOperation, Name: "記録の保存期間", Default: "3年"},
		{Section: SectionOperation, Name: "長期保存の記録の保存期間", Default: "5年"},
		{Section: SectionOperation, Name: "短期保存の記録の保存期間", Default: "1年"},
		{Section: SectionOperation, Name: "ログの保存期間", Default: "1年"},
		{Section: SectionOperation, Name: "記録の保管場所", Default: "このシステム（OpenGRC）"},
		{Section: SectionOperation, Name: "入退室記録の保管場所", Default: "入退室の記録簿"},
		{Section: SectionOperation, Name: "持ち出し記録の保管場所", Default: "持ち出しの記録簿"},
		{Section: SectionOperation, Name: "方針の周知の方法", Default: "従業員ポータルでの公開と、全員からの署名の取得"},
		{Section: SectionOperation, Name: "方針の掲示場所", Default: "従業員ポータル"},
		{Section: SectionOperation, Name: "年間計画を作る月", Default: "4月"},
		{Section: SectionOperation, Name: "教育の実施月", Default: "6月"},
		{Section: SectionOperation, Name: "理解度の合格基準", Default: "正答率80%以上"},
		{Section: SectionOperation, Name: "教育の受講率の目標", Default: "100%"},
		{Section: SectionOperation, Name: "未受講者の受講期限", Default: "1か月"},
		{Section: SectionOperation, Name: "是正処置の期限", Default: "1か月"},
		{Section: SectionOperation, Name: "運用の確認の頻度", Default: "毎月"},
		{Section: SectionOperation, Name: "取扱状況の確認の頻度", Default: "月1回"},
		{Section: SectionOperation, Name: "アクセス権の見直しの頻度", Default: "半年に1回"},
		{Section: SectionOperation, Name: "パスワードの最小文字数", Default: "12"},
		{Section: SectionOperation, Name: "脆弱性への対応期限", Default: "緊急度に応じて1週間から1か月"},
		{Section: SectionOperation, Name: "バックアップの頻度", Default: "毎日"},
		{Section: SectionOperation, Name: "復元テストの頻度", Default: "年1回"},
		{Section: SectionOperation, Name: "端末の確認の頻度", Default: "月1回"},
		{Section: SectionOperation, Name: "監査の事前通知", Default: "1週間"},
		{Section: SectionOperation, Name: "監査報告の期限", Default: "2週間"},
		{Section: SectionOperation, Name: "開示等の回答目標日数", Default: "14"},
		{Section: SectionOperation, Name: "苦情の一次回答の営業日数", Default: "3"},
		{Section: SectionOperation, Name: "委託先の事故を確認する年数", Default: "3"},
		{Section: SectionOperation, Name: "その他の受付方法", Default: "当社が指定するウェブフォーム"},
		{Section: SectionOperation, Name: "本人確認のオンラインの方法", Default: "ログインしたうえでの手続きや、ご登録の情報との照合"},
		{Section: SectionOperation, Name: "電磁的記録の形式", Default: "PDF または CSV のファイル（パスワードを付けてメールで送付するか、ダウンロード用の URL をお知らせします）"},
		{Section: SectionOperation, Name: "委託している業務", Default: "クラウドサービスによるデータの保管"},
		{Section: SectionOperation, Name: "審査機関への事故の報告先"},
		{Section: SectionOperation, Name: "審査機関への事故の報告期限"},
		{Section: SectionOperation, Name: "事故の報告の手引き"},

		// 選択に付随する値（選択によって文章に残ったときだけ必須）
		{Section: SectionChoice, Name: "第三者提供の内容"},
		{Section: SectionChoice, Name: "共同利用する項目"},
		{Section: SectionChoice, Name: "共同利用者の範囲"},
		{Section: SectionChoice, Name: "共同利用の目的"},
		{Section: SectionChoice, Name: "共同利用の責任者の名称"},
		{Section: SectionChoice, Name: "共同利用の責任者の住所"},
		{Section: SectionChoice, Name: "共同利用の責任者の代表者"},
		{Section: SectionChoice, Name: "外国提供の目的"},
		{Section: SectionChoice, Name: "提供先の国"},
		{Section: SectionChoice, Name: "外国の提供先の種類"},
		{Section: SectionChoice, Name: "保管する国"},
		{Section: SectionChoice, Name: "保管先の事業者の種類"},
		{Section: SectionChoice, Name: "認定個人情報保護団体の名称"},
		{Section: SectionChoice, Name: "苦情の申出先の名称"},
		{Section: SectionChoice, Name: "団体の住所"},
		{Section: SectionChoice, Name: "団体の電話番号"},
		{Section: SectionChoice, Name: "手数料の金額"},
		{Section: SectionChoice, Name: "手数料の支払方法"},
		{Section: SectionChoice, Name: "手数料の支払期限"},
	}

	// Choices lists the selections used by <!-- if 名前=値 --> blocks.
	Choices = []Choice{
		{Name: "第三者提供", Values: []string{"しない", "する"}, Default: "しない"},
		{Name: "共同利用", Values: []string{"しない", "する"}, Default: "しない"},
		{Name: "外国にある第三者への提供", Values: []string{"しない", "する"}, Default: "しない"},
		{Name: "外国でのデータの保管", Values: []string{"しない", "する"}, Default: "しない"},
		{Name: "Cookieなどの外部送信", Values: []string{"なし", "あり"}, Default: "なし"},
		{Name: "匿名加工情報・仮名加工情報", Values: []string{"扱わない", "扱う"}, Default: "扱わない"},
		{Name: "認定個人情報保護団体", Values: []string{"加入していない", "加入している"}, Default: "加入していない"},
		{Name: "手数料", Values: []string{"なし", "あり"}, Default: "なし"},
		{Name: "事務所の仕切り", Values: []string{"あり", "なし"}, Default: "あり"},
	}

	// DefaultPurposes is used when the configuration has no 利用目的.
	DefaultPurposes = []PurposeRow{
		{Category: "お客様・取引先の担当者", Items: "氏名、会社名、部署、役職、連絡先", Purposes: []string{"契約の締結と履行", "サービスの提供とサポート", "請求と支払", "各種のご連絡"}},
		{Category: "当社サービスの利用者", Items: "氏名、メールアドレス、ログイン情報、利用履歴", Purposes: []string{"アカウントの管理", "サービスの提供・改善", "不正利用の防止"}},
		{Category: "お問い合わせをいただいた方", Items: "氏名、連絡先、お問い合わせの内容", Purposes: []string{"お問い合わせへの回答と、そのための記録"}},
		{Category: "採用に応募された方", Items: "氏名、連絡先、学歴・職歴、応募書類の記載事項", Purposes: []string{"採用選考", "選考結果などのご連絡", "採用後の雇用管理の準備"}},
		{Category: "従業者と退職者", Items: "氏名、住所、連絡先、人事・給与に関する情報", Purposes: []string{"人事・労務・給与の管理", "法令に基づく手続き", "福利厚生", "社内の連絡"}},
		{Category: "委託を受けて取り扱う個人情報", Items: "委託元との契約で定めるもの", Purposes: []string{"委託元との契約で定める業務の遂行"}, Retained: new(false)},
	}
)

func variableByName(name string) (Variable, bool) {
	for _, v := range Variables {
		if v.Name == name {
			return v, true
		}
	}
	return Variable{}, false
}

func choiceByName(name string) (Choice, bool) {
	for _, c := range Choices {
		if c.Name == name {
			return c, true
		}
	}
	return Choice{}, false
}
