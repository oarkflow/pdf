package main

import (
	"fmt"
	"os"

	"github.com/oarkflow/pdf"
	"github.com/oarkflow/pdf/layout"
	"github.com/oarkflow/pdf/template"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "create":
		cmdCreate(os.Args[2:])
	case "merge":
		cmdMerge(os.Args[2:])
	case "info":
		cmdInfo(os.Args[2:])
	case "text":
		cmdText(os.Args[2:])
	case "sign":
		cmdSign(os.Args[2:])
	case "html":
		cmdHTML(os.Args[2:])
	case "version":
		fmt.Println("pdf version 0.1.0")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: pdf <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  create <output.pdf>                       Create a sample PDF")
	fmt.Println("  merge  <output.pdf> <input1.pdf> ...      Merge multiple PDFs")
	fmt.Println("  info   <file.pdf>                         Show PDF info")
	fmt.Println("  text   <file.pdf> [page]                  Extract text")
	fmt.Println("  sign   -key key.pem -cert cert.pem <in> <out>  Sign a PDF")
	fmt.Println("  html   <input.html> <output.pdf>          Convert HTML to PDF")
	fmt.Println("  version                                   Show version")
}

func cmdCreate(args []string) {
	output := "output.pdf"
	if len(args) > 0 {
		output = args[0]
	}

	t := template.New("sample")
	t.AddSection("main",
		layout.NewHeading(layout.H1, "Sample PDF Document"),
		layout.NewSpacer(12),
		layout.NewParagraph("This is a sample PDF created by the pdf CLI tool."),
		layout.NewSpacer(8),
		layout.NewParagraph("You can use this tool to create, merge, and manipulate PDF files."),
	)

	if err := t.Execute(nil, output); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating PDF: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created %s\n", output)
}

func cmdMerge(args []string) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: pdf merge <output.pdf> <input1.pdf> <input2.pdf> ...")
		os.Exit(1)
	}
	output := args[0]
	inputs := args[1:]

	if err := pdf.Merge(output, inputs...); err != nil {
		fmt.Fprintf(os.Stderr, "Error merging PDFs: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Merged %d files into %s\n", len(inputs), output)
}

func cmdInfo(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: pdf info <file.pdf>")
		os.Exit(1)
	}
	fmt.Printf("Info for %s: (PDF reader not yet implemented)\n", args[0])
}

func cmdText(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: pdf text <file.pdf> [page]")
		os.Exit(1)
	}
	fmt.Printf("Text extraction for %s: (PDF reader not yet implemented)\n", args[0])
}

func cmdSign(args []string) {
	if len(args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: pdf sign -key key.pem -cert cert.pem <input.pdf> <output.pdf>")
		os.Exit(1)
	}
	fmt.Println("PDF signing: (not yet implemented)")
}

func cmdHTML(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: pdf html <input.html> <output.pdf>")
		os.Exit(1)
	}
	input := args[0]
	output := args[1]

	htmlBytes, err := os.ReadFile(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", input, err)
		os.Exit(1)
	}

	if err := pdf.FromHTML(string(htmlBytes), output); err != nil {
		fmt.Fprintf(os.Stderr, "Error converting HTML to PDF: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Converted %s to %s\n", input, output)
}
