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

package xlsxexport_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"go.probo.inc/probo/pkg/xlsxexport"
)

func validSheet() xlsxexport.Sheet {
	return xlsxexport.Sheet{
		Name:     "台帳",
		Title:    "個人情報管理台帳",
		Subtitle: "サンプル株式会社",
		Columns: []xlsxexport.Column{
			{Header: "業務の名称", Width: 24},
			{Header: "備考"},
		},
		Rows: [][]string{
			{"採用", "=HYPERLINK(\"http://example.com\")"},
			{"顧客対応", "複数行\nの値"},
		},
	}
}

func openWorkbook(t *testing.T, data []byte) *excelize.File {
	t.Helper()

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	return f
}

func TestBuild_Layout(t *testing.T) {
	t.Parallel()

	data, err := xlsxexport.Build(validSheet())
	require.NoError(t, err)

	f := openWorkbook(t, data)

	t.Run(
		"writes title, subtitle, header and rows",
		func(t *testing.T) {
			t.Parallel()

			rows, err := f.GetRows("台帳")
			require.NoError(t, err)
			require.Len(t, rows, 6)

			assert.Equal(t, []string{"個人情報管理台帳"}, rows[0])
			assert.Equal(t, []string{"サンプル株式会社"}, rows[1])
			assert.Empty(t, rows[2])
			assert.Equal(t, []string{"業務の名称", "備考"}, rows[3])
			assert.Equal(t, []string{"顧客対応", "複数行\nの値"}, rows[5])
		},
	)

	t.Run(
		"stores formula-like text as a plain string",
		func(t *testing.T) {
			t.Parallel()

			formula, err := f.GetCellFormula("台帳", "B5")
			require.NoError(t, err)
			assert.Empty(t, formula)

			cellType, err := f.GetCellType("台帳", "B5")
			require.NoError(t, err)
			assert.NotEqual(t, excelize.CellTypeFormula, cellType)
		},
	)

	t.Run(
		"applies column widths",
		func(t *testing.T) {
			t.Parallel()

			width, err := f.GetColWidth("台帳", "A")
			require.NoError(t, err)
			assert.InDelta(t, 24, width, 0.01)

			width, err = f.GetColWidth("台帳", "B")
			require.NoError(t, err)
			assert.InDelta(t, 20, width, 0.01)
		},
	)

	t.Run(
		"prints on A4 landscape, one page wide",
		func(t *testing.T) {
			t.Parallel()

			layout, err := f.GetPageLayout("台帳")
			require.NoError(t, err)
			require.NotNil(t, layout.Size)
			require.NotNil(t, layout.Orientation)
			require.NotNil(t, layout.FitToWidth)
			assert.Equal(t, 9, *layout.Size)
			assert.Equal(t, "landscape", *layout.Orientation)
			assert.Equal(t, 1, *layout.FitToWidth)
		},
	)

	t.Run(
		"freezes the header row",
		func(t *testing.T) {
			t.Parallel()

			panes, err := f.GetPanes("台帳")
			require.NoError(t, err)
			assert.True(t, panes.Freeze)
			assert.Equal(t, 4, panes.YSplit)
		},
	)
}

func TestBuild_NoRows(t *testing.T) {
	t.Parallel()

	sheet := validSheet()
	sheet.Rows = nil

	data, err := xlsxexport.Build(sheet)
	require.NoError(t, err)

	rows, err := openWorkbook(t, data).GetRows("台帳")
	require.NoError(t, err)
	require.Len(t, rows, 4)
	assert.Equal(t, []string{"業務の名称", "備考"}, rows[3])
}

func TestBuild_InvalidSheet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*xlsxexport.Sheet)
	}{
		{"empty name", func(s *xlsxexport.Sheet) { s.Name = "" }},
		{"name too long", func(s *xlsxexport.Sheet) { s.Name = strings.Repeat("台", 32) }},
		{"name with invalid character", func(s *xlsxexport.Sheet) { s.Name = "台帳/2026" }},
		{"no columns", func(s *xlsxexport.Sheet) { s.Columns = nil }},
		{"row with wrong cell count", func(s *xlsxexport.Sheet) { s.Rows = [][]string{{"only one"}} }},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				t.Parallel()

				sheet := validSheet()
				tt.mutate(&sheet)

				_, err := xlsxexport.Build(sheet)
				assert.Error(t, err)
			},
		)
	}
}

func TestBuildWorkbook_MultipleSheets(t *testing.T) {
	t.Parallel()

	first := validSheet()
	second := validSheet()
	second.Name = "一覧"
	second.Title = "指摘事項一覧"
	second.Rows = [][]string{{"不適合", "記録なし"}}

	data, err := xlsxexport.BuildWorkbook(first, second)
	require.NoError(t, err)

	f := openWorkbook(t, data)
	assert.Equal(t, []string{"台帳", "一覧"}, f.GetSheetList())
	assert.Equal(t, 0, f.GetActiveSheetIndex())

	rows, err := f.GetRows("一覧")
	require.NoError(t, err)
	require.Len(t, rows, 5)
	assert.Equal(t, "指摘事項一覧", rows[0][0])
	assert.Equal(t, []string{"不適合", "記録なし"}, rows[4])

	panes, err := f.GetPanes("一覧")
	require.NoError(t, err)
	assert.True(t, panes.Freeze)
}

func TestBuildWorkbook_Invalid(t *testing.T) {
	t.Parallel()

	_, err := xlsxexport.BuildWorkbook()
	assert.Error(t, err)

	dup := validSheet()
	_, err = xlsxexport.BuildWorkbook(validSheet(), dup)
	assert.Error(t, err)

	bad := validSheet()
	bad.Columns = nil
	_, err = xlsxexport.BuildWorkbook(validSheet(), bad)
	assert.Error(t, err)
}
