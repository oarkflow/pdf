package main

import (
	"bufio"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/oarkflow/pdf"
	"github.com/oarkflow/pdf/converter"
	"github.com/oarkflow/pdf/core"
	"github.com/oarkflow/pdf/layout"
	"github.com/oarkflow/pdf/reader"
	"github.com/oarkflow/pdf/sign"
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
	case "split":
		cmdSplit(os.Args[2:])
	case "delete-pages":
		cmdDeletePages(os.Args[2:])
	case "reorder":
		cmdReorder(os.Args[2:])
	case "rotate":
		cmdRotate(os.Args[2:])
	case "protect":
		cmdProtect(os.Args[2:])
	case "decrypt":
		cmdDecrypt(os.Args[2:])
	case "watermark":
		cmdWatermark(os.Args[2:])
	case "page-numbers":
		cmdPageNumbers(os.Args[2:])
	case "set-metadata":
		cmdSetMetadata(os.Args[2:])
	case "extract-images":
		cmdExtractImages(os.Args[2:])
	case "text":
		cmdText(os.Args[2:])
	case "to-text":
		cmdText(os.Args[2:])
	case "to-html":
		cmdToHTML(os.Args[2:])
	case "to-markdown":
		cmdToMarkdown(os.Args[2:])
	case "to-json":
		cmdToJSON(os.Args[2:])
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
	fmt.Println("  info   [options] <file.pdf>               Show PDF info")
	fmt.Println("  split  [options] <file.pdf>               Extract selected pages")
	fmt.Println("  delete-pages [options] <file.pdf>         Remove selected pages")
	fmt.Println("  reorder [options] <file.pdf>              Reorder pages")
	fmt.Println("  rotate [options] <file.pdf>               Rotate selected pages")
	fmt.Println("  protect [options] <file.pdf>              Encrypt a PDF")
	fmt.Println("  decrypt [options] <file.pdf>              Remove PDF encryption")
	fmt.Println("  watermark [options] <file.pdf>            Add a text watermark")
	fmt.Println("  page-numbers [options] <file.pdf>         Add page numbers")
	fmt.Println("  set-metadata [options] <file.pdf>         Replace document metadata")
	fmt.Println("  extract-images [options] <file.pdf>       Extract embedded images")
	fmt.Println("  text   [options] <file.pdf>               Extract PDF text")
	fmt.Println("  to-text [options] <file.pdf>              Extract PDF text")
	fmt.Println("  to-html [options] <file.pdf>              Convert PDF to HTML")
	fmt.Println("  to-markdown [options] <file.pdf>          Convert PDF to Markdown")
	fmt.Println("  to-json [options] <file.pdf>              Convert PDF to structured JSON")
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
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	password := fs.String("password", "", "PDF password")
	promptPassword := fs.Bool("prompt-password", false, "prompt for the PDF password")
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: pdf info [options] <file.pdf>")
		fs.PrintDefaults()
		os.Exit(1)
	}
	pass := *password
	if *promptPassword && pass == "" {
		pass = promptPDFPassword()
	}
	info, err := pdf.Info(fs.Arg(0), pass)
	if err != nil && pass == "" && isPasswordError(err) {
		pass = promptPDFPassword()
		info, err = pdf.Info(fs.Arg(0), pass)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading info: %v\n", err)
		os.Exit(1)
	}
	if *jsonOut {
		writeJSON(os.Stdout, info)
		return
	}
	fmt.Printf("File: %s\n", info.Path)
	fmt.Printf("Pages: %d\n", info.Pages)
	fmt.Printf("Encrypted: %t\n", info.Encrypted)
	if len(info.Metadata) > 0 {
		fmt.Println("Metadata:")
		for k, v := range info.Metadata {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}
	if len(info.Outlines) > 0 {
		fmt.Println("Outlines:")
		printOutlines(info.Outlines, "  ")
	}
	if len(info.Annotations) > 0 {
		fmt.Println("Annotations:")
		for _, a := range info.Annotations {
			fmt.Printf("  page %d: %s", a.Page, a.Subtype)
			if a.URI != "" {
				fmt.Printf(" %s", a.URI)
			}
			if a.Content != "" {
				fmt.Printf(" %s", a.Content)
			}
			fmt.Println()
		}
	}
	if len(info.PageSizes) > 0 {
		fmt.Println("Page sizes:")
		for _, page := range info.PageSizes {
			fmt.Printf("  %d: %.0fx%.0f", page.Number, page.Width, page.Height)
			if page.Rotation != 0 {
				fmt.Printf(" rotate=%d", page.Rotation)
			}
			fmt.Println()
		}
	}
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

func cmdToMarkdown(args []string) {
	fs := flag.NewFlagSet("to-markdown", flag.ExitOnError)
	output := fs.String("o", "", "write output to file")
	page := fs.Int("page", 0, "1-based page number to convert")
	pageRange := fs.String("pages", "", "1-based page range, e.g. 1-3,5")
	password := fs.String("password", "", "PDF password")
	promptPassword := fs.Bool("prompt-password", false, "prompt for the PDF password")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: pdf to-markdown [options] <file.pdf>")
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
	markdown, err := pdf.ToMarkdown(fs.Arg(0), opts)
	if err != nil && opts.Password == "" && isPasswordError(err) {
		opts.Password = promptPDFPassword()
		markdown, err = pdf.ToMarkdown(fs.Arg(0), opts)
	}
	writeStringOutput("Markdown", markdown, *output, err)
}

func cmdToJSON(args []string) {
	fs := flag.NewFlagSet("to-json", flag.ExitOnError)
	output := fs.String("o", "", "write output to file")
	page := fs.Int("page", 0, "1-based page number to convert")
	pageRange := fs.String("pages", "", "1-based page range, e.g. 1-3,5")
	password := fs.String("password", "", "PDF password")
	promptPassword := fs.Bool("prompt-password", false, "prompt for the PDF password")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: pdf to-json [options] <file.pdf>")
		fs.PrintDefaults()
		os.Exit(1)
	}
	opts := converter.ConvertOptions{Password: *password, ExtractImages: true, DetectTables: true}
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
	data, err := pdf.ToJSON(fs.Arg(0), opts)
	if err != nil && opts.Password == "" && isPasswordError(err) {
		opts.Password = promptPDFPassword()
		data, err = pdf.ToJSON(fs.Arg(0), opts)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting PDF to JSON: %v\n", err)
		os.Exit(1)
	}
	if *output != "" {
		if err := os.WriteFile(*output, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", *output, err)
			os.Exit(1)
		}
		fmt.Printf("Wrote JSON to %s\n", *output)
		return
	}
	os.Stdout.Write(data)
	fmt.Println()
}

func cmdSplit(args []string) {
	fs := flag.NewFlagSet("split", flag.ExitOnError)
	output := fs.String("o", "", "output PDF path")
	page := fs.Int("page", 0, "1-based page number to extract")
	pageRange := fs.String("pages", "", "1-based page range, e.g. 1-3,5")
	password := fs.String("password", "", "PDF password")
	promptPassword := fs.Bool("prompt-password", false, "prompt for the PDF password")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 || *output == "" {
		fmt.Fprintln(os.Stderr, "Usage: pdf split -o output.pdf [options] <file.pdf>")
		fs.PrintDefaults()
		os.Exit(1)
	}
	var pages []int
	if *page > 0 {
		pages = []int{*page - 1}
	} else if *pageRange != "" {
		var err error
		pages, err = parsePages(*pageRange)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid page range: %v\n", err)
			os.Exit(1)
		}
	}
	opts := converter.ConvertOptions{Password: *password}
	if *promptPassword && opts.Password == "" {
		opts.Password = promptPDFPassword()
	}
	err := pdf.Split(fs.Arg(0), *output, pages, opts)
	if err != nil && opts.Password == "" && isPasswordError(err) {
		opts.Password = promptPDFPassword()
		err = pdf.Split(fs.Arg(0), *output, pages, opts)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error splitting PDF: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Wrote PDF to %s\n", *output)
}

func cmdExtractImages(args []string) {
	fs := flag.NewFlagSet("extract-images", flag.ExitOnError)
	outputDir := fs.String("o", ".", "output directory")
	page := fs.Int("page", 0, "1-based page number")
	pageRange := fs.String("pages", "", "1-based page range, e.g. 1-3,5")
	password := fs.String("password", "", "PDF password")
	promptPassword := fs.Bool("prompt-password", false, "prompt for the PDF password")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: pdf extract-images -o images <file.pdf>")
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
	images, err := pdf.ExtractImages(fs.Arg(0), opts)
	if err != nil && opts.Password == "" && isPasswordError(err) {
		opts.Password = promptPDFPassword()
		images, err = pdf.ExtractImages(fs.Arg(0), opts)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting images: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", *outputDir, err)
		os.Exit(1)
	}
	for i, img := range images {
		name := converter.ImageFilename(img, i)
		path := filepath.Join(*outputDir, name)
		if err := os.WriteFile(path, img.Data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", path, err)
			os.Exit(1)
		}
	}
	fmt.Printf("Extracted %d image(s) to %s\n", len(images), *outputDir)
}

func cmdDeletePages(args []string) {
	fs := flag.NewFlagSet("delete-pages", flag.ExitOnError)
	output := fs.String("o", "", "output PDF path")
	pageRange := fs.String("pages", "", "1-based pages to delete, e.g. 1-3,5")
	password := fs.String("password", "", "PDF password")
	promptPassword := fs.Bool("prompt-password", false, "prompt for the PDF password")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 || *output == "" || *pageRange == "" {
		fmt.Fprintln(os.Stderr, "Usage: pdf delete-pages -pages 2,4 -o output.pdf <file.pdf>")
		fs.PrintDefaults()
		os.Exit(1)
	}
	pages, err := parsePages(*pageRange)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid page range: %v\n", err)
		os.Exit(1)
	}
	opts := converter.ConvertOptions{Password: passwordValue(*password, *promptPassword)}
	err = pdf.DeletePages(fs.Arg(0), *output, pages, opts)
	if err != nil && opts.Password == "" && isPasswordError(err) {
		opts.Password = promptPDFPassword()
		err = pdf.DeletePages(fs.Arg(0), *output, pages, opts)
	}
	exitOnPDFError("deleting pages", err)
	fmt.Printf("Wrote PDF to %s\n", *output)
}

func cmdReorder(args []string) {
	fs := flag.NewFlagSet("reorder", flag.ExitOnError)
	output := fs.String("o", "", "output PDF path")
	order := fs.String("pages", "", "1-based output page order, e.g. 3,1,2")
	password := fs.String("password", "", "PDF password")
	promptPassword := fs.Bool("prompt-password", false, "prompt for the PDF password")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 || *output == "" || *order == "" {
		fmt.Fprintln(os.Stderr, "Usage: pdf reorder -pages 3,1,2 -o output.pdf <file.pdf>")
		fs.PrintDefaults()
		os.Exit(1)
	}
	pages, err := parsePages(*order)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid page order: %v\n", err)
		os.Exit(1)
	}
	opts := converter.ConvertOptions{Password: passwordValue(*password, *promptPassword)}
	err = pdf.Reorder(fs.Arg(0), *output, pages, opts)
	if err != nil && opts.Password == "" && isPasswordError(err) {
		opts.Password = promptPDFPassword()
		err = pdf.Reorder(fs.Arg(0), *output, pages, opts)
	}
	exitOnPDFError("reordering pages", err)
	fmt.Printf("Wrote PDF to %s\n", *output)
}

func cmdRotate(args []string) {
	fs := flag.NewFlagSet("rotate", flag.ExitOnError)
	output := fs.String("o", "", "output PDF path")
	pageRange := fs.String("pages", "", "1-based pages to rotate, default all")
	rotation := fs.Int("degrees", 90, "rotation: 0, 90, 180, or 270")
	password := fs.String("password", "", "PDF password")
	promptPassword := fs.Bool("prompt-password", false, "prompt for the PDF password")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 || *output == "" {
		fmt.Fprintln(os.Stderr, "Usage: pdf rotate -degrees 90 -o output.pdf <file.pdf>")
		fs.PrintDefaults()
		os.Exit(1)
	}
	var pages []int
	if *pageRange != "" {
		var err error
		pages, err = parsePages(*pageRange)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid page range: %v\n", err)
			os.Exit(1)
		}
	}
	opts := converter.ConvertOptions{Password: passwordValue(*password, *promptPassword)}
	err := pdf.Rotate(fs.Arg(0), *output, pages, *rotation, opts)
	if err != nil && opts.Password == "" && isPasswordError(err) {
		opts.Password = promptPDFPassword()
		err = pdf.Rotate(fs.Arg(0), *output, pages, *rotation, opts)
	}
	exitOnPDFError("rotating PDF", err)
	fmt.Printf("Wrote PDF to %s\n", *output)
}

func cmdProtect(args []string) {
	fs := flag.NewFlagSet("protect", flag.ExitOnError)
	output := fs.String("o", "", "output PDF path")
	userPassword := fs.String("user-password", "", "user/open password")
	ownerPassword := fs.String("owner-password", "", "owner password")
	inputPassword := fs.String("password", "", "input PDF password")
	promptInput := fs.Bool("prompt-password", false, "prompt for the input PDF password")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 || *output == "" {
		fmt.Fprintln(os.Stderr, "Usage: pdf protect -user-password secret -o protected.pdf <file.pdf>")
		fs.PrintDefaults()
		os.Exit(1)
	}
	if *userPassword == "" {
		fmt.Fprintln(os.Stderr, "Missing -user-password")
		os.Exit(1)
	}
	owner := *ownerPassword
	if owner == "" {
		owner = *userPassword
	}
	opts := converter.ConvertOptions{Password: passwordValue(*inputPassword, *promptInput)}
	err := pdf.Protect(fs.Arg(0), *output, core.EncryptionConfig{
		Algorithm:     core.AES_128,
		UserPassword:  *userPassword,
		OwnerPassword: owner,
		Permissions:   0xFFFFF0C4,
	}, opts)
	if err != nil && opts.Password == "" && isPasswordError(err) {
		opts.Password = promptPDFPassword()
		err = pdf.Protect(fs.Arg(0), *output, core.EncryptionConfig{
			Algorithm:     core.AES_128,
			UserPassword:  *userPassword,
			OwnerPassword: owner,
			Permissions:   0xFFFFF0C4,
		}, opts)
	}
	exitOnPDFError("protecting PDF", err)
	fmt.Printf("Wrote protected PDF to %s\n", *output)
}

func cmdDecrypt(args []string) {
	fs := flag.NewFlagSet("decrypt", flag.ExitOnError)
	output := fs.String("o", "", "output PDF path")
	password := fs.String("password", "", "PDF password")
	promptPassword := fs.Bool("prompt-password", false, "prompt for the PDF password")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 || *output == "" {
		fmt.Fprintln(os.Stderr, "Usage: pdf decrypt -prompt-password -o unlocked.pdf <file.pdf>")
		fs.PrintDefaults()
		os.Exit(1)
	}
	opts := converter.ConvertOptions{Password: passwordValue(*password, *promptPassword)}
	err := pdf.Decrypt(fs.Arg(0), *output, opts)
	if err != nil && opts.Password == "" && isPasswordError(err) {
		opts.Password = promptPDFPassword()
		err = pdf.Decrypt(fs.Arg(0), *output, opts)
	}
	exitOnPDFError("decrypting PDF", err)
	fmt.Printf("Wrote decrypted PDF to %s\n", *output)
}

func cmdWatermark(args []string) {
	fs := flag.NewFlagSet("watermark", flag.ExitOnError)
	output := fs.String("o", "", "output PDF path")
	text := fs.String("text", "", "watermark text")
	pageRange := fs.String("pages", "", "1-based pages, default all")
	fontSize := fs.Float64("size", 48, "font size")
	angle := fs.Float64("angle", 35, "angle in degrees")
	password := fs.String("password", "", "PDF password")
	promptPassword := fs.Bool("prompt-password", false, "prompt for the PDF password")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 || *output == "" || *text == "" {
		fmt.Fprintln(os.Stderr, "Usage: pdf watermark -text DRAFT -o output.pdf <file.pdf>")
		fs.PrintDefaults()
		os.Exit(1)
	}
	var pages []int
	if *pageRange != "" {
		var err error
		pages, err = parsePages(*pageRange)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid page range: %v\n", err)
			os.Exit(1)
		}
	}
	err := pdf.Watermark(fs.Arg(0), *output, pdf.WatermarkOptions{
		Text:     *text,
		FontSize: *fontSize,
		Angle:    *angle,
		Pages:    pages,
		Password: passwordValue(*password, *promptPassword),
	})
	exitOnPDFError("adding watermark", err)
	fmt.Printf("Wrote watermarked PDF to %s\n", *output)
}

func cmdPageNumbers(args []string) {
	fs := flag.NewFlagSet("page-numbers", flag.ExitOnError)
	output := fs.String("o", "", "output PDF path")
	format := fs.String("format", "Page %d of %d", "fmt format with page and total")
	fontSize := fs.Float64("size", 10, "font size")
	margin := fs.Float64("margin", 36, "bottom margin")
	password := fs.String("password", "", "PDF password")
	promptPassword := fs.Bool("prompt-password", false, "prompt for the PDF password")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 || *output == "" {
		fmt.Fprintln(os.Stderr, "Usage: pdf page-numbers -o output.pdf <file.pdf>")
		fs.PrintDefaults()
		os.Exit(1)
	}
	err := pdf.AddPageNumbers(fs.Arg(0), *output, pdf.PageNumberOptions{
		Format:   *format,
		FontSize: *fontSize,
		Margin:   *margin,
		Password: passwordValue(*password, *promptPassword),
	})
	exitOnPDFError("adding page numbers", err)
	fmt.Printf("Wrote numbered PDF to %s\n", *output)
}

func cmdSetMetadata(args []string) {
	fs := flag.NewFlagSet("set-metadata", flag.ExitOnError)
	output := fs.String("o", "", "output PDF path")
	title := fs.String("title", "", "title")
	author := fs.String("author", "", "author")
	subject := fs.String("subject", "", "subject")
	keywords := fs.String("keywords", "", "keywords")
	creator := fs.String("creator", "", "creator")
	producer := fs.String("producer", "", "producer")
	password := fs.String("password", "", "PDF password")
	promptPassword := fs.Bool("prompt-password", false, "prompt for the PDF password")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 || *output == "" {
		fmt.Fprintln(os.Stderr, "Usage: pdf set-metadata -title Title -o output.pdf <file.pdf>")
		fs.PrintDefaults()
		os.Exit(1)
	}
	metadata := map[string]string{}
	if *title != "" {
		metadata["Title"] = *title
	}
	if *author != "" {
		metadata["Author"] = *author
	}
	if *subject != "" {
		metadata["Subject"] = *subject
	}
	if *keywords != "" {
		metadata["Keywords"] = *keywords
	}
	if *creator != "" {
		metadata["Creator"] = *creator
	}
	if *producer != "" {
		metadata["Producer"] = *producer
	}
	opts := converter.ConvertOptions{Password: passwordValue(*password, *promptPassword)}
	err := pdf.SetMetadata(fs.Arg(0), *output, metadata, opts)
	if err != nil && opts.Password == "" && isPasswordError(err) {
		opts.Password = promptPDFPassword()
		err = pdf.SetMetadata(fs.Arg(0), *output, metadata, opts)
	}
	exitOnPDFError("setting metadata", err)
	fmt.Printf("Wrote PDF to %s\n", *output)
}

func cmdSign(args []string) {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	keyPath := fs.String("key", "", "private key PEM path")
	certPath := fs.String("cert", "", "certificate PEM path")
	output := fs.String("o", "", "output PDF path")
	reason := fs.String("reason", "", "signing reason")
	location := fs.String("location", "", "signing location")
	name := fs.String("name", "", "signer name")
	contact := fs.String("contact", "", "signer contact")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 || *output == "" || *keyPath == "" || *certPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: pdf sign -key key.pem -cert cert.pem -o signed.pdf <file.pdf>")
		fs.PrintDefaults()
		os.Exit(1)
	}
	signer, err := loadLocalSigner(*keyPath, *certPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading signer: %v\n", err)
		os.Exit(1)
	}
	err = sign.SignFile(fs.Arg(0), *output, sign.Options{
		Signer:      signer,
		Reason:      *reason,
		Location:    *location,
		Name:        *name,
		ContactInfo: *contact,
	})
	exitOnPDFError("signing PDF", err)
	fmt.Printf("Wrote signed PDF to %s\n", *output)
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

func writeStringOutput(label, value, output string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting PDF to %s: %v\n", strings.ToLower(label), err)
		os.Exit(1)
	}
	if output != "" {
		if err := os.WriteFile(output, []byte(value), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", output, err)
			os.Exit(1)
		}
		fmt.Printf("Wrote %s to %s\n", label, output)
		return
	}
	fmt.Print(value)
	if !strings.HasSuffix(value, "\n") {
		fmt.Println()
	}
}

func writeJSON(w io.Writer, v interface{}) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func printOutlines(items []reader.OutlineInfo, indent string) {
	for _, item := range items {
		if item.Page > 0 {
			fmt.Printf("%s- %s (page %d)\n", indent, item.Title, item.Page)
		} else {
			fmt.Printf("%s- %s\n", indent, item.Title)
		}
		if len(item.Children) > 0 {
			printOutlines(item.Children, indent+"  ")
		}
	}
}

func loadLocalSigner(keyPath, certPath string) (*sign.LocalSigner, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	key, err := parsePrivateKeyPEM(keyData)
	if err != nil {
		return nil, err
	}
	certs, err := parseCertificatesPEM(certData)
	if err != nil {
		return nil, err
	}
	signer := sign.NewLocalSigner(key, certs)
	if signer == nil {
		return nil, fmt.Errorf("certificate chain is empty")
	}
	return signer, nil
}

func parsePrivateKeyPEM(data []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM private key found")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return cryptoSignerFromKey(key)
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	return nil, fmt.Errorf("unsupported private key PEM type %q", block.Type)
}

func cryptoSignerFromKey(key interface{}) (crypto.Signer, error) {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return k, nil
	case *ecdsa.PrivateKey:
		return k, nil
	case crypto.Signer:
		return k, nil
	default:
		return nil, fmt.Errorf("unsupported private key type %T", key)
	}
}

func parseCertificatesPEM(data []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("no PEM certificates found")
	}
	return certs, nil
}

func passwordValue(password string, prompt bool) string {
	if prompt && password == "" {
		return promptPDFPassword()
	}
	return password
}

func exitOnPDFError(action string, err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "Error %s: %v\n", action, err)
	os.Exit(1)
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
