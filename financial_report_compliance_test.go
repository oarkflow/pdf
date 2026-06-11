package pdf

import (
	"bytes"
	"strings"
	"testing"

	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/layout"
	"github.com/oarkflow/pdf/template"
)

func TestFinancialReportComplianceIncludesTaggedContent(t *testing.T) {
	pdfa4 := document.PDFA4
	pdfua2 := document.PDFUA2
	data := template.FinancialReportData{
		Title:    "Compliance Report",
		PageSize: document.A4,
		Margins:  document.Margins{Top: 36, Right: 36, Bottom: 42, Left: 36},
		PDFA:     &pdfa4,
		PDFUA:    &pdfua2,
		Language: "en-US",
		Blocks: []template.FinancialReportBlock{
			{Heading: &template.FinancialReportHeading{
				Text:      "Compliance Report",
				FontSize:  16,
				Bold:      true,
				Align:     layout.AlignCenter,
				TextColor: [3]float64{0, 0, 0},
				Height:    24,
			}},
			{Table: &template.FinancialReportTable{
				ColumnWidths: []float64{120, 180},
				Rows: [][]template.FinancialReportCell{{
					{Text: "Metric", Bold: true},
					{Text: "Value"},
				}},
			}},
		},
	}

	compiled, err := template.CompileFinancialReport(data)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := compiled.WriteStreamingTo(&out); err != nil {
		t.Fatal(err)
	}
	pdf := out.String()
	for _, want := range []string{
		"<pdfaid:part>4</pdfaid:part>",
		"<pdfuaid:part>2</pdfuaid:part>",
		"/StructTreeRoot",
		"/StructParents 0",
		"/ParentTree",
		"/S /TD",
		"/S /P",
		"/MarkInfo",
		"/Lang (en-US)",
	} {
		if !strings.Contains(pdf, want) {
			t.Fatalf("expected compliant PDF to contain %q", want)
		}
	}
}
