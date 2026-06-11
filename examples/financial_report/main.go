package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/layout"
	"github.com/oarkflow/pdf/template"
)

func main() {
	iterations := envInt("FINANCIAL_REPORT_ITERATIONS", 5000)
	workers := envInt("FINANCIAL_REPORT_WORKERS", 96)

	dataDir := os.Getenv("FINANCIAL_REPORT_DATA_DIR")
	if dataDir == "" {
		dataDir = "/tmp/gopdfsuit/sampledata/financialreport/data"
	}
	barChart, err := os.ReadFile(filepath.Join(dataDir, "bar_chart.png"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading bar_chart.png: %v\n", err)
		os.Exit(1)
	}
	pieChart, err := os.ReadFile(filepath.Join(dataDir, "pie_chart.png"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading pie_chart.png: %v\n", err)
		os.Exit(1)
	}
	data := buildFinancialReportData(barChart, pieChart)
	compiled, err := template.CompileFinancialReport(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compile failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== oarkflow/pdf Native Financial Report Benchmark ===")
	fmt.Printf("OS: %s, Arch: %s, NumCPU: %d, GoVersion: %s\n", runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), runtime.Version())
	fmt.Printf("Running %d iterations using %d workers...\n\n", iterations, workers)

	var warmPDF bytes.Buffer
	warmPDF.Grow(compiled.EstimatedSize())
	if err := compiled.WriteStreamingTo(&warmPDF); err != nil {
		fmt.Fprintf(os.Stderr, "Warm-up serialization failed: %v\n", err)
		os.Exit(1)
	}

	jobs := make(chan int, iterations)
	results := make(chan time.Duration, iterations)
	errors := make(chan error, iterations)
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				start := time.Now()
				var out bytes.Buffer
				out.Grow(compiled.EstimatedSize())
				err := compiled.WriteStreamingTo(&out)
				if err == nil && out.Len() == 0 {
					err = fmt.Errorf("empty PDF output")
				}
				elapsed := time.Since(start)
				if err != nil {
					errors <- err
					continue
				}
				results <- elapsed
			}
		}()
	}

	totalStart := time.Now()
	for i := 1; i <= iterations; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	totalTime := time.Since(totalStart)
	close(results)
	close(errors)

	if len(errors) > 0 {
		fmt.Printf("Encountered %d errors during execution.\n", len(errors))
		os.Exit(1)
	}

	var durations []time.Duration
	var sum time.Duration
	for d := range results {
		durations = append(durations, d)
		sum += d
	}
	if len(durations) == 0 {
		fmt.Println("No results collected.")
		return
	}
	minDuration, maxDuration := durations[0], durations[0]
	for _, d := range durations {
		if d < minDuration {
			minDuration = d
		}
		if d > maxDuration {
			maxDuration = d
		}
	}
	avgDuration := sum / time.Duration(len(durations))
	opsPerSec := float64(iterations) / totalTime.Seconds()

	fmt.Println("=== Performance Summary ===")
	fmt.Printf("  Iterations:    %d\n", iterations)
	fmt.Printf("  Concurrency:   %d workers\n", workers)
	fmt.Printf("  Total time:    %.3f s\n", totalTime.Seconds())
	fmt.Printf("  Throughput:    %.2f ops/sec\n", opsPerSec)
	fmt.Println()
	fmt.Printf("  Avg Latency:   %.3f ms\n", float64(avgDuration.Microseconds())/1000.0)
	fmt.Printf("  Min Latency:   %.3f ms\n", float64(minDuration.Microseconds())/1000.0)
	fmt.Printf("  Max Latency:   %.3f ms\n", float64(maxDuration.Microseconds())/1000.0)
	fmt.Printf("  PDF size:      %d bytes (%.2f KB)\n", warmPDF.Len(), float64(warmPDF.Len())/1024.0)
}

func envInt(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func buildFinancialReportData(barChart, pieChart []byte) template.FinancialReportData {
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
						exampleCell("Company Name:", true, layout.AlignLeft, &shade), exampleCell("TechCorp Industries Inc.", false, layout.AlignLeft, &shade),
						exampleCell("Report Period:", true, layout.AlignLeft, &shade), exampleCell("Q4 2025", false, layout.AlignLeft, &shade),
					},
					{
						exampleCell("Address:", true, layout.AlignLeft, nil), exampleCell("123 Business Ave, Suite 456, City, State 12345", false, layout.AlignLeft, nil),
						exampleCell("Fiscal Year:", true, layout.AlignLeft, nil), exampleCell("2025", false, layout.AlignLeft, nil),
					},
				},
			}},
			{Heading: &template.FinancialReportHeading{Text: "SECTION B: FINANCIAL SUMMARY", FontSize: 12, Bold: true, TextColor: white, BgColor: &blue, Height: 22}},
			{Table: &template.FinancialReportTable{
				ColumnWidths: []float64{348.85, 174.43},
				CellPadding:  4,
				BorderWidth:  0.4,
				Rows: [][]template.FinancialReportCell{
					exampleSummaryRow("Total Revenue", "$2,450,000", false, nil),
					exampleSummaryRow("Cost of Goods Sold", "$1,225,000", false, &shade),
					exampleSummaryRow("Gross Profit", "$1,225,000", true, &highlight),
					exampleSummaryRow("Operating Expenses", "$750,000", false, nil),
					exampleSummaryRow("Net Income", "$125,000", true, &highlight),
					exampleSummaryRow("Total Assets", "$5,000,000", true, &highlight),
					exampleSummaryRow("Total Liabilities", "$2,500,000", false, nil),
					exampleSummaryRow("Shareholders' Equity", "$2,500,000", true, &highlight),
				},
			}},
			{Spacer: 160},
			{Heading: &template.FinancialReportHeading{Text: "SECTION C: CHARTS", FontSize: 12, Bold: true, TextColor: white, BgColor: &blue, Height: 22}},
			{Table: &template.FinancialReportTable{
				ColumnWidths: []float64{261.64, 261.64},
				CellPadding:  4,
				BorderWidth:  0.4,
				Rows: [][]template.FinancialReportCell{{
					exampleCell("REVENUE BREAKDOWN", true, layout.AlignCenter, &shade),
					exampleCell("EXPENSE DISTRIBUTION", true, layout.AlignCenter, &shade),
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
		},
	}
}

func exampleCell(text string, bold bool, align layout.Alignment, bg *[3]float64) template.FinancialReportCell {
	return template.FinancialReportCell{Text: text, Bold: bold, Align: align, Bg: bg, FontSize: 10}
}

func exampleSummaryRow(label, value string, bold bool, bg *[3]float64) []template.FinancialReportCell {
	return []template.FinancialReportCell{
		exampleCell(label, bold, layout.AlignLeft, bg),
		exampleCell(value, bold, layout.AlignRight, bg),
	}
}
