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

// Package xlsxexport renders a single-sheet register (a title, a subtitle and
// a table) as an .xlsx workbook that prints on A4 landscape. It is the shared
// base for the Japanese PrivacyMark audit forms.
package xlsxexport

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	// ContentType is the media type of the produced workbook.
	ContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

	// FontFamily is the default font of Japanese Excel, so the workbook
	// looks native on Windows and macOS.
	FontFamily = "游ゴシック"

	maxSheetNameLength = 31
	defaultColumnWidth = 20
	titleRow           = 1
	subtitleRow        = 2
	headerRow          = 4
	firstDataRow       = headerRow + 1
	paperSizeA4        = 9
)

type (
	Column struct {
		Header string
		// Width is in Excel character units. Zero means a default width.
		Width float64
	}

	Sheet struct {
		Name     string
		Title    string
		Subtitle string
		Columns  []Column
		Rows     [][]string
	}

	styles struct {
		title    int
		subtitle int
		header   int
		data     int
	}
)

const invalidSheetNameChars = `:\/?*[]`

func (s Sheet) validate() error {
	if s.Name == "" {
		return errors.New("sheet name is required")
	}

	if len([]rune(s.Name)) > maxSheetNameLength {
		return fmt.Errorf("sheet name must be at most %d characters", maxSheetNameLength)
	}

	if strings.ContainsAny(s.Name, invalidSheetNameChars) {
		return fmt.Errorf("sheet name must not contain any of %q", invalidSheetNameChars)
	}

	if len(s.Columns) == 0 {
		return errors.New("at least one column is required")
	}

	for i, row := range s.Rows {
		if len(row) != len(s.Columns) {
			return fmt.Errorf("row %d has %d cells, want %d", i, len(row), len(s.Columns))
		}
	}

	return nil
}

// Build renders the sheet and returns the workbook bytes. Every value is
// written as a string cell, so text that starts with "=" is never evaluated
// as a formula.
func Build(s Sheet) ([]byte, error) {
	return BuildWorkbook(s)
}

// BuildWorkbook renders the sheets, in order, into one workbook and returns
// its bytes. The first sheet is the one shown when the file is opened.
func BuildWorkbook(sheets ...Sheet) ([]byte, error) {
	if len(sheets) == 0 {
		return nil, errors.New("invalid workbook: at least one sheet is required")
	}

	names := make(map[string]struct{}, len(sheets))
	for i, s := range sheets {
		if err := s.validate(); err != nil {
			return nil, fmt.Errorf("invalid sheet %d: %w", i, err)
		}

		key := strings.ToLower(s.Name)
		if _, ok := names[key]; ok {
			return nil, fmt.Errorf("invalid sheet %d: duplicate sheet name %q", i, s.Name)
		}
		names[key] = struct{}{}
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	if err := f.SetSheetName("Sheet1", sheets[0].Name); err != nil {
		return nil, fmt.Errorf("cannot name sheet: %w", err)
	}

	for _, s := range sheets[1:] {
		if _, err := f.NewSheet(s.Name); err != nil {
			return nil, fmt.Errorf("cannot add sheet: %w", err)
		}
	}

	st, err := newStyles(f)
	if err != nil {
		return nil, err
	}

	steps := []func(*excelize.File, Sheet, styles) error{
		writeHeading,
		writeTable,
		setLayout,
	}
	for _, s := range sheets {
		for _, step := range steps {
			if err := step(f, s, st); err != nil {
				return nil, err
			}
		}
	}

	f.SetActiveSheet(0)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("cannot write workbook: %w", err)
	}

	return buf.Bytes(), nil
}

func newStyles(f *excelize.File) (styles, error) {
	border := []excelize.Border{
		{Type: "left", Color: "808080", Style: 1},
		{Type: "top", Color: "808080", Style: 1},
		{Type: "right", Color: "808080", Style: 1},
		{Type: "bottom", Color: "808080", Style: 1},
	}

	defs := []*excelize.Style{
		{Font: &excelize.Font{Family: FontFamily, Size: 14, Bold: true}},
		{Font: &excelize.Font{Family: FontFamily, Size: 10}},
		{
			Font:      &excelize.Font{Family: FontFamily, Size: 10, Bold: true},
			Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"D9E1F2"}},
			Border:    border,
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		},
		{
			Font:      &excelize.Font{Family: FontFamily, Size: 10},
			Border:    border,
			Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
		},
	}

	ids := make([]int, len(defs))
	for i, def := range defs {
		id, err := f.NewStyle(def)
		if err != nil {
			return styles{}, fmt.Errorf("cannot create style: %w", err)
		}
		ids[i] = id
	}

	return styles{title: ids[0], subtitle: ids[1], header: ids[2], data: ids[3]}, nil
}

func writeHeading(f *excelize.File, s Sheet, st styles) error {
	cells := []struct {
		row   int
		value string
		style int
	}{
		{titleRow, s.Title, st.title},
		{subtitleRow, s.Subtitle, st.subtitle},
	}

	for _, c := range cells {
		ref := cellName(1, c.row)
		if err := f.SetCellStr(s.Name, ref, c.value); err != nil {
			return fmt.Errorf("cannot write heading: %w", err)
		}

		if err := f.SetCellStyle(s.Name, ref, ref, c.style); err != nil {
			return fmt.Errorf("cannot style heading: %w", err)
		}
	}

	return nil
}

func writeTable(f *excelize.File, s Sheet, st styles) error {
	for i, col := range s.Columns {
		ref := cellName(i+1, headerRow)
		if err := f.SetCellStr(s.Name, ref, col.Header); err != nil {
			return fmt.Errorf("cannot write header: %w", err)
		}

		width := col.Width
		if width <= 0 {
			width = defaultColumnWidth
		}

		name := columnName(i + 1)
		if err := f.SetColWidth(s.Name, name, name, width); err != nil {
			return fmt.Errorf("cannot set column width: %w", err)
		}
	}

	lastCol := len(s.Columns)
	if err := f.SetCellStyle(s.Name, cellName(1, headerRow), cellName(lastCol, headerRow), st.header); err != nil {
		return fmt.Errorf("cannot style header: %w", err)
	}

	for r, row := range s.Rows {
		for c, value := range row {
			if err := f.SetCellStr(s.Name, cellName(c+1, firstDataRow+r), value); err != nil {
				return fmt.Errorf("cannot write cell: %w", err)
			}
		}
	}

	if len(s.Rows) > 0 {
		lastRow := firstDataRow + len(s.Rows) - 1
		if err := f.SetCellStyle(s.Name, cellName(1, firstDataRow), cellName(lastCol, lastRow), st.data); err != nil {
			return fmt.Errorf("cannot style data: %w", err)
		}
	}

	return nil
}

func setLayout(f *excelize.File, s Sheet, _ styles) error {
	lastCol := len(s.Columns)
	lastRow := max(headerRow, firstDataRow+len(s.Rows)-1)
	headerRef := fmt.Sprintf("%s:%s", cellName(1, headerRow), cellName(lastCol, lastRow))

	if err := f.AutoFilter(s.Name, headerRef, nil); err != nil {
		return fmt.Errorf("cannot set auto filter: %w", err)
	}

	if err := f.SetPanes(s.Name, &excelize.Panes{
		Freeze:      true,
		YSplit:      headerRow,
		TopLeftCell: cellName(1, firstDataRow),
		ActivePane:  "bottomLeft",
	}); err != nil {
		return fmt.Errorf("cannot freeze header: %w", err)
	}

	size, orientation, fitWidth, fitHeight := paperSizeA4, "landscape", 1, 0
	if err := f.SetPageLayout(s.Name, &excelize.PageLayoutOptions{
		Size:        &size,
		Orientation: &orientation,
		FitToWidth:  &fitWidth,
		FitToHeight: &fitHeight,
	}); err != nil {
		return fmt.Errorf("cannot set page layout: %w", err)
	}

	fitToPage := true
	if err := f.SetSheetProps(s.Name, &excelize.SheetPropsOptions{FitToPage: &fitToPage}); err != nil {
		return fmt.Errorf("cannot set fit to page: %w", err)
	}

	if err := f.SetDefinedName(&excelize.DefinedName{
		Name:     "_xlnm.Print_Titles",
		RefersTo: fmt.Sprintf("'%s'!$%d:$%d", s.Name, headerRow, headerRow),
		Scope:    s.Name,
	}); err != nil {
		return fmt.Errorf("cannot repeat header on each page: %w", err)
	}

	if err := f.SetHeaderFooter(s.Name, &excelize.HeaderFooterOptions{
		OddFooter: "&C&P / &N",
	}); err != nil {
		return fmt.Errorf("cannot set footer: %w", err)
	}

	return nil
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

func columnName(col int) string {
	name, _ := excelize.ColumnNumberToName(col)
	return name
}
