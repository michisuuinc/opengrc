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

package probo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
)

func ledgerTestActivity(name string) *coredata.ProcessingActivity {
	return &coredata.ProcessingActivity{
		ID:                    gid.New(gid.NewTenantID(), coredata.ProcessingActivityEntityType),
		Name:                  name,
		SpecialOrCriminalData: coredata.ProcessingActivitySpecialOrCriminalDatumNo,
	}
}

func ledgerRow(t *testing.T, data ledgerData) map[string]string {
	t.Helper()

	sheet := buildLedgerSheet(data, time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))
	require.Len(t, sheet.Rows, 1)

	row := make(map[string]string, len(sheet.Columns))
	for i, c := range sheet.Columns {
		row[c.Header] = sheet.Rows[0][i]
	}

	return row
}

func TestBuildLedgerSheet_Heading(t *testing.T) {
	t.Parallel()

	sheet := buildLedgerSheet(
		ledgerData{OrganizationName: "サンプル株式会社"},
		time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC),
	)

	assert.Equal(t, "個人情報管理台帳", sheet.Name)
	assert.Equal(t, "個人情報管理台帳", sheet.Title)
	assert.Equal(t, "サンプル株式会社　出力日：2026-10-15", sheet.Subtitle)
	assert.Equal(t, "No.", sheet.Columns[0].Header)
	assert.Len(t, sheet.Columns, len(ledgerColumns)+1)
	assert.Empty(t, sheet.Rows)
}

func TestBuildLedgerSheet_Rows(t *testing.T) {
	t.Parallel()

	t.Run(
		"fills processing activity and ledger items",
		func(t *testing.T) {
			t.Parallel()

			a := ledgerTestActivity("採用選考")
			a.Purpose = new("応募者の選考と連絡")
			a.SpecialOrCriminalData = coredata.ProcessingActivitySpecialOrCriminalDatumPossible
			a.InternationalTransfers = true
			a.TransferSafeguard = new(coredata.ProcessingActivityTransferSafeguardAdequacyDecision)
			a.LastReviewDate = new(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))

			row := ledgerRow(t, ledgerData{
				Activities: []*coredata.ProcessingActivity{a},
				Details: map[gid.GID]*coredata.ProcessingActivityLedgerDetails{
					a.ID: {
						Department:      new("人事部"),
						StorageLocation: new("採用管理システム、書庫A"),
					},
				},
				ThirdPartyNames: map[gid.GID][]string{a.ID: {"採用代行A社", "クラウドB社"}},
			})

			assert.Equal(t, "1", row["No."])
			assert.Equal(t, "採用選考", row["業務の名称"])
			assert.Equal(t, "人事部", row["担当部門"])
			assert.Equal(t, "応募者の選考と連絡", row["利用目的"])
			assert.Equal(t, "可能性あり", row["要配慮個人情報"])
			assert.Equal(t, "採用管理システム、書庫A", row["保管の場所と媒体"])
			assert.Equal(t, "採用代行A社\nクラウドB社", row["委託先"])
			assert.Equal(t, "あり（十分性認定）", row["外国への移転"])
			assert.Equal(t, "2026-04-01", row["最終見直し日"])
			assert.Empty(t, row["次回見直し日"])
			assert.Empty(t, row["廃棄の方法"])
		},
	)

	t.Run(
		"leaves ledger items blank when the activity has none",
		func(t *testing.T) {
			t.Parallel()

			a := ledgerTestActivity("問い合わせ対応")

			row := ledgerRow(t, ledgerData{Activities: []*coredata.ProcessingActivity{a}})

			assert.Equal(t, "問い合わせ対応", row["業務の名称"])
			assert.Equal(t, "なし", row["要配慮個人情報"])
			assert.Equal(t, "なし", row["委託先"])
			assert.Equal(t, "なし", row["外国への移転"])
			for _, header := range []string{"担当部門", "おおよその件数", "取得の方法", "保管の場所と媒体", "アクセスできる人の範囲", "廃棄の方法"} {
				assert.Empty(t, row[header], header)
			}
		},
	)
}

func TestLedgerTransferLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "なし", ledgerTransferLabel(false, new(coredata.ProcessingActivityTransferSafeguardDerogations)))
	assert.Equal(t, "あり", ledgerTransferLabel(true, nil))
	assert.Equal(t, "あり（標準契約条項）", ledgerTransferLabel(true, new(coredata.ProcessingActivityTransferSafeguardStandardContractualClauses)))
}

func TestApplyLedgerDetailsUpdate(t *testing.T) {
	t.Parallel()

	original := coredata.ProcessingActivityLedgerDetails{
		Department:     new("総務部"),
		DisposalMethod: new("シュレッダー"),
	}

	var cleared *string
	updated := applyLedgerDetailsUpdate(original, &UpdateProcessingActivityLedgerDetailsRequest{
		Department:     new(new("人事部")),
		DisposalMethod: &cleared,
	})

	require.NotNil(t, updated.Department)
	assert.Equal(t, "人事部", *updated.Department)
	assert.Nil(t, updated.DisposalMethod)
	assert.Nil(t, updated.AccessScope)
	assert.Equal(t, "総務部", *original.Department, "original must not change")
}
