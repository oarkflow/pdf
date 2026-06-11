package pdf

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/layout"
	"github.com/oarkflow/pdf/template"
)

func BenchmarkNativeFinancialReport(b *testing.B) {
	barChart, pieChart := loadFinancialReportCharts(b)
	data := benchmarkFinancialReportData(barChart, pieChart)
	compiled, err := template.CompileFinancialReport(data)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var out bytes.Buffer
		out.Grow(compiled.EstimatedSize())
		if err := compiled.WriteStreamingTo(&out); err != nil {
			b.Fatal(err)
		}
		if out.Len() == 0 {
			b.Fatal("empty PDF output")
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "pdfs/s")
}

func BenchmarkNativeFinancialReportCompliance(b *testing.B) {
	barChart, pieChart := loadFinancialReportCharts(b)
	data := benchmarkFinancialReportData(barChart, pieChart)
	pdfa4 := document.PDFA4
	pdfua2 := document.PDFUA2
	data.PDFA = &pdfa4
	data.PDFUA = &pdfua2
	data.Language = "en-US"

	compiled, err := template.CompileFinancialReport(data)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var out bytes.Buffer
		out.Grow(compiled.EstimatedSize())
		if err := compiled.WriteStreamingTo(&out); err != nil {
			b.Fatal(err)
		}
		if out.Len() == 0 {
			b.Fatal("empty PDF output")
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "pdfs/s")
}

func loadFinancialReportCharts(tb testing.TB) ([]byte, []byte) {
	tb.Helper()

	dataDir := os.Getenv("FINANCIAL_REPORT_DATA_DIR")
	if dataDir == "" {
		dataDir = "/tmp/gopdfsuit/sampledata/financialreport/data"
	}

	barChart, err := os.ReadFile(filepath.Join(dataDir, "bar_chart.png"))
	if err != nil {
		tb.Skipf("financial report chart data not found: %v", err)
	}
	pieChart, err := os.ReadFile(filepath.Join(dataDir, "pie_chart.png"))
	if err != nil {
		tb.Skipf("financial report chart data not found: %v", err)
	}
	return barChart, pieChart
}

func benchmarkFinancialReportData(barChart, pieChart []byte) template.FinancialReportData {
	blue := [3]float64{0.129, 0.380, 0.549}
	darkBlue := [3]float64{0.082, 0.263, 0.376}
	white := [3]float64{1, 1, 1}
	shade := [3]float64{0.957, 0.965, 0.969}
	highlight := [3]float64{0.831, 0.902, 0.945}
	muted := [3]float64{0.337, 0.404, 0.451}

	return template.FinancialReportData{
		Title:      "FINANCIAL REPORT",
		Subject:    "Financial Report",
		PageSize:   document.A4,
		Margins:    document.Margins{Top: 36, Right: 36, Bottom: 42, Left: 36},
		FooterText: "TECHCORP INDUSTRIES INC. | FINANCIAL REPORT Q4 2025 | CONFIDENTIAL",
		Blocks: []template.FinancialReportBlock{
			{Heading: &template.FinancialReportHeading{Text: "FINANCIAL REPORT", FontSize: 24, Bold: true, Align: layout.AlignCenter, TextColor: white, BgColor: &darkBlue, Height: 44}},
			{Heading: &template.FinancialReportHeading{Text: "SECTION A: COMPANY INFORMATION", FontSize: 12, Bold: true, TextColor: white, BgColor: &blue, Height: 22}},
			{Table: &template.FinancialReportTable{
				ColumnWidths: []float64{105, 200, 105, 113.28},
				CellPadding:  4,
				BorderWidth:  0.4,
				Rows: [][]template.FinancialReportCell{
					{
						reportCell("Company Name:", true, layout.AlignLeft, &shade), reportCell("TechCorp Industries Inc.", false, layout.AlignLeft, &shade),
						reportCell("Report Period:", true, layout.AlignLeft, &shade), reportCell("Q4 2025", false, layout.AlignLeft, &shade),
					},
					{
						reportCell("Address:", true, layout.AlignLeft, nil), reportCell("123 Business Ave, Suite 456, City, State 12345", false, layout.AlignLeft, nil),
						reportCell("Fiscal Year:", true, layout.AlignLeft, nil), reportCell("2025", false, layout.AlignLeft, nil),
					},
					{
						reportCell("", false, layout.AlignLeft, &shade), reportCell("Go to Financial Summary", false, layout.AlignLeft, &shade),
						reportCell("Go to Charts", false, layout.AlignLeft, &shade), reportCell("", false, layout.AlignLeft, &shade),
					},
				},
			}},
			{Heading: &template.FinancialReportHeading{Text: "SECTION B: FINANCIAL SUMMARY", FontSize: 12, Bold: true, TextColor: white, BgColor: &blue, Height: 22}},
			{Table: &template.FinancialReportTable{
				ColumnWidths: []float64{348.85, 174.43},
				CellPadding:  4,
				BorderWidth:  0.4,
				Rows: [][]template.FinancialReportCell{
					summaryRow("Total Revenue", "$2,450,000", false, nil),
					summaryRow("Cost of Goods Sold", "$1,225,000", false, &shade),
					summaryRow("Gross Profit", "$1,225,000", true, &highlight),
					summaryRow("Operating Expenses", "$750,000", false, nil),
					summaryRow("Research & Development", "$200,000", false, &shade),
					summaryRow("Marketing & Sales", "$150,000", false, nil),
					summaryRow("Administrative Expenses", "$100,000", false, &shade),
					summaryRow("Depreciation & Amortization", "$50,000", false, nil),
					summaryRow("Interest Expense", "$25,000", false, &shade),
					summaryRow("Taxes", "$75,000", false, nil),
					summaryRow("Net Income", "$125,000", true, &highlight),
					summaryRow("Earnings Per Share", "$2.50", false, nil),
					summaryRow("Total Assets", "$5,000,000", true, &highlight),
					summaryRow("Total Liabilities", "$2,500,000", false, nil),
					summaryRow("Shareholders' Equity", "$2,500,000", true, &highlight),
				},
			}},
			{Spacer: 160},
			{Heading: &template.FinancialReportHeading{Text: "SECTION C: CHARTS", FontSize: 12, Bold: true, TextColor: white, BgColor: &blue, Height: 22}},
			{Table: &template.FinancialReportTable{
				ColumnWidths: []float64{261.64, 261.64},
				CellPadding:  4,
				BorderWidth:  0.4,
				Rows: [][]template.FinancialReportCell{{
					reportCell("REVENUE BREAKDOWN", true, layout.AlignCenter, &shade),
					reportCell("EXPENSE DISTRIBUTION", true, layout.AlignCenter, &shade),
				}},
			}},
			{ImageGrid: &template.FinancialReportImageGrid{
				ColumnWidths: []float64{261.64, 261.64},
				CellPadding:  4,
				BorderWidth:  0.4,
				Images: []template.FinancialReportImage{
					{Data: barChart, Alt: "BarChart", Width: 200, Height: 200, Align: layout.AlignCenter},
					{Data: pieChart, Alt: "PieChart", Width: 200, Height: 200, Align: layout.AlignCenter},
				},
				Captions: []template.FinancialReportCell{
					{Text: "Figure 1: Quarterly revenue comparison by region", FontSize: 8, Align: layout.AlignCenter, TextColor: muted},
					{Text: "Figure 2: Breakdown of operating expenses", FontSize: 8, Align: layout.AlignCenter, TextColor: muted},
				},
			}},
			{Spacer: 50},
		},
	}
}

func reportCell(text string, bold bool, align layout.Alignment, bg *[3]float64) template.FinancialReportCell {
	return template.FinancialReportCell{Text: text, Bold: bold, Align: align, Bg: bg, FontSize: 10}
}

func summaryRow(label, value string, bold bool, bg *[3]float64) []template.FinancialReportCell {
	return []template.FinancialReportCell{
		reportCell(label, bold, layout.AlignLeft, bg),
		reportCell(value, bold, layout.AlignRight, bg),
	}
}
