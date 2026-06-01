package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/oarkflow/pdf"
	"github.com/oarkflow/pdf/converter"
	"github.com/oarkflow/pdf/layout"
	"github.com/oarkflow/pdf/template"
	"golang.org/x/term"
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
	case "to-text":
		cmdText(os.Args[2:])
	case "to-html":
		cmdToHTML(os.Args[2:])
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
	fmt.Println("  text   [options] <file.pdf>               Extract PDF text")
	fmt.Println("  to-text [options] <file.pdf>              Extract PDF text")
	fmt.Println("  to-html [options] <file.pdf>              Convert PDF to HTML")
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
	fs := flag.NewFlagSet("text", flag.ExitOnError)
	output := fs.String("o", "", "write output to file")
	page := fs.Int("page", 0, "1-based page number to extract")
	pageRange := fs.String("pages", "", "1-based page range, e.g. 1-3,5")
	password := fs.String("password", "", "PDF password")
	promptPassword := fs.Bool("prompt-password", false, "prompt for the PDF password")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: pdf text [options] <file.pdf>")
		fs.PrintDefaults()
		os.Exit(1)
	}

	opts := converter.ConvertOptions{Password: *password}
	if *promptPassword && opts.Password == "" {
		opts.Password = promptPDFPassword()
	}
	if *page > 0 {
		opts.Pages = []int{*page - 1}
	} else if *pageRange != "" {
		pages, err := parsePages(*pageRange)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid page range: %v\n", err)
			os.Exit(1)
		}
		opts.Pages = pages
	}

	text, err := pdf.ToText(fs.Arg(0), opts)
	if err != nil && opts.Password == "" && isPasswordError(err) {
		opts.Password = promptPDFPassword()
		text, err = pdf.ToText(fs.Arg(0), opts)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting text: %v\n", err)
		os.Exit(1)
	}
	if *output != "" {
		if err := os.WriteFile(*output, []byte(text), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", *output, err)
			os.Exit(1)
		}
		fmt.Printf("Wrote text to %s\n", *output)
		return
	}
	fmt.Print(text)
	if !strings.HasSuffix(text, "\n") {
		fmt.Println()
	}
}

func cmdToHTML(args []string) {
	fs := flag.NewFlagSet("to-html", flag.ExitOnError)
	output := fs.String("o", "", "write output to file")
	mode := fs.String("mode", "reflowed", "conversion mode: reflowed or positioned")
	page := fs.Int("page", 0, "1-based page number to convert")
	pageRange := fs.String("pages", "", "1-based page range, e.g. 1-3,5")
	password := fs.String("password", "", "PDF password")
	promptPassword := fs.Bool("prompt-password", false, "prompt for the PDF password")
	extractImages := fs.Bool("images", true, "extract images")
	detectTables := fs.Bool("tables", true, "detect tables")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: pdf to-html [options] <file.pdf>")
		fs.PrintDefaults()
		os.Exit(1)
	}
	if *mode != "reflowed" && *mode != "positioned" {
		fmt.Fprintln(os.Stderr, "Invalid mode: use reflowed or positioned")
		os.Exit(1)
	}

	opts := converter.ConvertOptions{
		Mode:          *mode,
		ExtractImages: *extractImages,
		DetectTables:  *detectTables,
		Password:      *password,
	}
	if *promptPassword && opts.Password == "" {
		opts.Password = promptPDFPassword()
	}
	if *page > 0 {
		opts.Pages = []int{*page - 1}
	} else if *pageRange != "" {
		pages, err := parsePages(*pageRange)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid page range: %v\n", err)
			os.Exit(1)
		}
		opts.Pages = pages
	}

	html, err := pdf.ToHTML(fs.Arg(0), opts)
	if err != nil && opts.Password == "" && isPasswordError(err) {
		opts.Password = promptPDFPassword()
		html, err = pdf.ToHTML(fs.Arg(0), opts)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting PDF to HTML: %v\n", err)
		os.Exit(1)
	}
	if *output != "" {
		if err := os.WriteFile(*output, []byte(html), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", *output, err)
			os.Exit(1)
		}
		fmt.Printf("Wrote HTML to %s\n", *output)
		return
	}
	fmt.Print(html)
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

func parsePages(spec string) ([]int, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}
	seen := make(map[int]bool)
	var pages []int
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			start, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
			if err != nil || start < 1 {
				return nil, fmt.Errorf("invalid start page %q", bounds[0])
			}
			end, err := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err != nil || end < start {
				return nil, fmt.Errorf("invalid end page %q", bounds[1])
			}
			for p := start; p <= end; p++ {
				idx := p - 1
				if !seen[idx] {
					seen[idx] = true
					pages = append(pages, idx)
				}
			}
			continue
		}
		p, err := strconv.Atoi(part)
		if err != nil || p < 1 {
			return nil, fmt.Errorf("invalid page %q", part)
		}
		idx := p - 1
		if !seen[idx] {
			seen[idx] = true
			pages = append(pages, idx)
		}
	}
	return pages, nil
}

func isPasswordError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalid password") || strings.Contains(msg, "encrypted pdf")
}

func promptPDFPassword() string {
	fmt.Fprint(os.Stderr, "PDF password: ")
	if term.IsTerminal(int(os.Stdin.Fd())) {
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
			os.Exit(1)
		}
		return string(password)
	}

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
		os.Exit(1)
	}
	return strings.TrimRight(line, "\r\n")
}
