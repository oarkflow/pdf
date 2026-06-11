package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oarkflow/pdf"
	"github.com/oarkflow/pdf/converter"
	"github.com/oarkflow/pdf/core"
	"github.com/oarkflow/pdf/layout"
	"github.com/oarkflow/pdf/sign"
	"github.com/oarkflow/pdf/template"
	"github.com/urfave/cli/v3"
)

func runCLI(args []string) error {
	return buildCLI().Run(context.Background(), args)
}

func buildCLI() *cli.Command {
	return &cli.Command{
		Name:                  "pdf",
		Usage:                 "PDF creation, conversion, inspection, and manipulation",
		UsageText:             "pdf <command> [options] [arguments]",
		Version:               "0.1.0",
		EnableShellCompletion: true,
		Suggest:               true,
		Commands: []*cli.Command{
			createCommand(),
			htmlCommand(),
			imagesToPDFCommand(),
			mergeCommand(),
			infoCommand(),
			validateCommand(),
			searchCommand(),
			textCommand(),
			toHTMLCommand(),
			toMarkdownCommand(),
			toJSONCommand(),
			extractImagesCommand(),
			splitCommand(),
			deletePagesCommand(),
			reorderCommand(),
			rotateCommand(),
			passwordCommand(),
			protectCommand(),
			decryptCommand(),
			signCommand(),
			watermarkCommand(),
			pageNumbersCommand(),
			setMetadataCommand(),
		},
	}
}

func createCommand() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a sample PDF",
		ArgsUsage: "[output.pdf]",
		Category:  "Create and combine",
		Action: func(_ context.Context, cmd *cli.Command) error {
			output := cmd.Args().First()
			if output == "" {
				output = "output.pdf"
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
				return fmt.Errorf("creating PDF: %w", err)
			}
			fmt.Printf("Created %s\n", output)
			return nil
		},
	}
}

func htmlCommand() *cli.Command {
	return &cli.Command{
		Name:      "html",
		Usage:     "Convert HTML to PDF",
		ArgsUsage: "<input.html> <output.pdf>",
		Category:  "Create and combine",
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 2 {
				return usageError("pdf html <input.html> <output.pdf>")
			}
			htmlBytes, err := os.ReadFile(cmd.Args().Get(0))
			if err != nil {
				return fmt.Errorf("reading %s: %w", cmd.Args().Get(0), err)
			}
			if err := pdf.FromHTML(string(htmlBytes), cmd.Args().Get(1)); err != nil {
				return fmt.Errorf("converting HTML: %w", err)
			}
			fmt.Printf("Converted %s to %s\n", cmd.Args().Get(0), cmd.Args().Get(1))
			return nil
		},
	}
}

func imagesToPDFCommand() *cli.Command {
	return &cli.Command{
		Name:      "images-to-pdf",
		Aliases:   []string{"image-pdf"},
		Usage:     "Convert images into one PDF",
		ArgsUsage: "[image1 image2 ...]",
		Category:  "Create and combine",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Usage: "output PDF path", TakesFile: true, Required: true},
			&cli.StringSliceFlag{Name: "input", Usage: "image input path; repeatable", TakesFile: true},
			&cli.StringSliceFlag{Name: "dir", Usage: "image input directory; repeatable", TakesFile: true},
			&cli.StringFlag{Name: "size", Value: "A4", Usage: "page size: A3, A4, A5, Letter, Legal"},
			&cli.FloatFlag{Name: "width", Usage: "custom page width in points"},
			&cli.FloatFlag{Name: "height", Usage: "custom page height in points"},
			&cli.FloatFlag{Name: "margin", Usage: "page margin in points"},
			&cli.StringFlag{Name: "fit", Value: "contain", Usage: "image fit: contain, cover, fill, none"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			var inputs []pdf.MergeInput
			for _, path := range cmd.StringSlice("input") {
				inputs = append(inputs, pdf.MergeInput{Path: path, Type: "image"})
			}
			for _, dir := range cmd.StringSlice("dir") {
				inputs = append(inputs, pdf.MergeInput{Directory: dir, Type: "image"})
			}
			for _, path := range cmd.Args().Slice() {
				inputs = append(inputs, pdf.MergeInput{Path: path, Type: "image"})
			}
			if len(inputs) == 0 {
				return usageError("pdf images-to-pdf -o output.pdf image1.png image2.jpg")
			}
			err := pdf.ImagesToPDF(pdf.ImagePDFOptions{
				Output: cmd.String("o"),
				Page: pdf.MergePageOptions{
					Size:     cmd.String("size"),
					Width:    cmd.Float("width"),
					Height:   cmd.Float("height"),
					Margin:   cmd.Float("margin"),
					ImageFit: cmd.String("fit"),
				},
				Inputs: inputs,
			})
			if err != nil {
				return fmt.Errorf("converting images to PDF: %w", err)
			}
			fmt.Printf("Wrote image PDF to %s\n", cmd.String("o"))
			return nil
		},
	}
}

func mergeCommand() *cli.Command {
	return &cli.Command{
		Name:        "merge",
		Usage:       "Merge PDFs and images",
		ArgsUsage:   "[output.pdf input1 input2 ...]",
		Category:    "Create and combine",
		Description: "Merge explicit PDF/image inputs, whole directories, or an ordered JSON config.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Usage: "JSON merge config", TakesFile: true},
			&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Usage: "output PDF path", TakesFile: true},
			&cli.StringSliceFlag{Name: "input", Usage: "input PDF/image path; repeatable", TakesFile: true},
			&cli.StringSliceFlag{Name: "dir", Usage: "input directory; repeatable", TakesFile: true},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if config := cmd.String("config"); config != "" {
				if err := pdf.MergeWithConfig(config); err != nil {
					return fmt.Errorf("merging files: %w", err)
				}
				fmt.Printf("Merged files from config %s\n", config)
				return nil
			}
			output := cmd.String("o")
			positional := cmd.Args().Slice()
			var inputs []pdf.MergeInput
			for _, path := range cmd.StringSlice("input") {
				inputs = append(inputs, pdf.MergeInput{Path: path})
			}
			for _, dir := range cmd.StringSlice("dir") {
				inputs = append(inputs, pdf.MergeInput{Directory: dir})
			}
			if output == "" && len(positional) >= 2 {
				output = positional[0]
				positional = positional[1:]
			}
			for _, path := range positional {
				inputs = append(inputs, pdf.MergeInput{Path: path})
			}
			if output == "" || len(inputs) == 0 {
				return usageError("pdf merge [-config merge.json] [-o output.pdf] [-input file] [-dir directory] [output.pdf input1 input2 ...]")
			}
			if err := pdf.MergeWithOptions(pdf.MergeOptions{Output: output, Inputs: inputs}); err != nil {
				return fmt.Errorf("merging files: %w", err)
			}
			fmt.Printf("Merged inputs into %s\n", output)
			return nil
		},
	}
}

func infoCommand() *cli.Command {
	return &cli.Command{
		Name:      "info",
		Usage:     "Show PDF info",
		ArgsUsage: "<file.pdf>",
		Category:  "Inspect and extract",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "password", Usage: "PDF password"},
			&cli.BoolFlag{Name: "prompt-password", Usage: "prompt for the PDF password"},
			&cli.BoolFlag{Name: "json", Usage: "print JSON"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return usageError("pdf info [options] <file.pdf>")
			}
			pass := passwordValue(cmd.String("password"), cmd.Bool("prompt-password"))
			info, err := pdf.Info(cmd.Args().First(), pass)
			if err != nil && pass == "" && isPasswordError(err) {
				pass = promptPDFPassword()
				info, err = pdf.Info(cmd.Args().First(), pass)
			}
			if err != nil {
				return fmt.Errorf("reading info: %w", err)
			}
			if cmd.Bool("json") {
				writeJSON(os.Stdout, info)
				return nil
			}
			printInfo(info)
			return nil
		},
	}
}

func validateCommand() *cli.Command {
	return &cli.Command{
		Name:      "validate",
		Usage:     "Validate PDF structure and compliance profiles",
		ArgsUsage: "<file.pdf>",
		Category:  "Inspect and extract",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "password", Usage: "PDF password"},
			&cli.BoolFlag{Name: "prompt-password", Usage: "prompt for the PDF password"},
			&cli.StringSliceFlag{Name: "profile", Aliases: []string{"p"}, Usage: "compliance profile: pdf, auto, pdfa-1b, pdfa-2b, pdfa-4, pdfua-1, pdfua-2, pdfx-*, pdfe-*, pdfvt-*, pades-*"},
			&cli.BoolFlag{Name: "strict", Usage: "treat warnings as validation failures"},
			&cli.StringFlag{Name: "external", Usage: "external validator adapter, e.g. verapdf"},
			&cli.BoolFlag{Name: "json", Usage: "print JSON"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return usageError("pdf validate [options] <file.pdf>")
			}
			profiles := complianceProfilesFromCLI(cmd.StringSlice("profile"))
			result := pdf.ValidateCompliance(cmd.Args().First(), pdf.ComplianceOptions{
				Profiles: profiles,
				Password: passwordValue(cmd.String("password"), cmd.Bool("prompt-password")),
				Strict:   cmd.Bool("strict"),
				External: cmd.String("external"),
			})
			if cmd.Bool("json") {
				writeJSON(os.Stdout, result)
				if !result.Valid {
					return fmt.Errorf("PDF validation failed")
				}
				return nil
			}
			fmt.Printf("File: %s\n", result.Path)
			fmt.Printf("Valid: %t\n", result.Valid)
			fmt.Printf("Encrypted: %t\n", result.Encrypted)
			fmt.Printf("Pages: %d\n", result.Pages)
			if len(result.Profiles) > 0 {
				fmt.Printf("Profiles: %s\n", joinComplianceProfiles(result.Profiles))
			}
			if len(result.DetectedProfiles) > 0 {
				fmt.Printf("Detected: %s\n", joinComplianceProfiles(result.DetectedProfiles))
			}
			if result.Error != "" {
				fmt.Printf("Error: %s\n", result.Error)
			}
			if len(result.Issues) > 0 {
				printComplianceIssues(result.Issues)
			}
			if !result.Valid {
				return fmt.Errorf("PDF validation failed")
			}
			return nil
		},
	}
}

func complianceProfilesFromCLI(values []string) []pdf.ComplianceProfile {
	if len(values) == 0 {
		return []pdf.ComplianceProfile{pdf.ProfilePDF}
	}
	profiles := make([]pdf.ComplianceProfile, 0, len(values))
	for _, value := range values {
		profiles = append(profiles, pdf.ComplianceProfile(value))
	}
	return profiles
}

func joinComplianceProfiles(profiles []pdf.ComplianceProfile) string {
	parts := make([]string, len(profiles))
	for i, profile := range profiles {
		parts[i] = string(profile)
	}
	return strings.Join(parts, ", ")
}

func printComplianceIssues(issues []pdf.ComplianceIssue) {
	current := ""
	for _, issue := range issues {
		if issue.Severity != current {
			current = issue.Severity
			fmt.Printf("%s:\n", strings.Title(current))
		}
		prefix := "  -"
		if issue.Profile != "" {
			prefix += " [" + string(issue.Profile) + "]"
		}
		if issue.Clause != "" {
			prefix += " " + issue.Clause + ":"
		}
		fmt.Printf("%s %s\n", prefix, issue.Message)
		if issue.Suggestion != "" {
			fmt.Printf("    Suggestion: %s\n", issue.Suggestion)
		}
	}
}

func searchCommand() *cli.Command {
	return &cli.Command{
		Name:      "search",
		Usage:     "Search text in a PDF",
		ArgsUsage: "<file.pdf>",
		Category:  "Inspect and extract",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "query", Aliases: []string{"q"}, Usage: "text query", Required: true},
			&cli.BoolFlag{Name: "case-sensitive", Usage: "case-sensitive search"},
			&cli.StringFlag{Name: "password", Usage: "PDF password"},
			&cli.BoolFlag{Name: "prompt-password", Usage: "prompt for the PDF password"},
			&cli.BoolFlag{Name: "json", Usage: "print JSON"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return usageError("pdf search -q text <file.pdf>")
			}
			opts := pdf.SearchOptions{
				Query:         cmd.String("query"),
				CaseSensitive: cmd.Bool("case-sensitive"),
				Password:      passwordValue(cmd.String("password"), cmd.Bool("prompt-password")),
			}
			matches, err := pdf.SearchText(cmd.Args().First(), opts)
			if err != nil && opts.Password == "" && isPasswordError(err) {
				opts.Password = promptPDFPassword()
				matches, err = pdf.SearchText(cmd.Args().First(), opts)
			}
			if err != nil {
				return fmt.Errorf("searching PDF: %w", err)
			}
			if cmd.Bool("json") {
				writeJSON(os.Stdout, matches)
				return nil
			}
			for _, match := range matches {
				fmt.Printf("page %d, line %d, col %d: %s\n", match.Page, match.Line, match.Column, match.Context)
			}
			fmt.Printf("Found %d match(es)\n", len(matches))
			return nil
		},
	}
}

func textCommand() *cli.Command {
	return convertTextLikeCommand("text", []string{"to-text"}, "Extract PDF text", "Text", func(path string, opts converter.ConvertOptions) (string, error) {
		return pdf.ToText(path, opts)
	})
}

func toMarkdownCommand() *cli.Command {
	return convertTextLikeCommand("to-markdown", nil, "Convert PDF to Markdown", "Markdown", func(path string, opts converter.ConvertOptions) (string, error) {
		return pdf.ToMarkdown(path, opts)
	})
}

func convertTextLikeCommand(name string, aliases []string, usage, label string, fn func(string, converter.ConvertOptions) (string, error)) *cli.Command {
	return &cli.Command{
		Name:      name,
		Aliases:   aliases,
		Usage:     usage,
		ArgsUsage: "<file.pdf>",
		Category:  "Inspect and extract",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Usage: "write output to file", TakesFile: true},
			&cli.IntFlag{Name: "page", Usage: "1-based page number"},
			&cli.StringFlag{Name: "pages", Usage: "1-based page range, e.g. 1-3,5"},
			&cli.StringFlag{Name: "password", Usage: "PDF password"},
			&cli.BoolFlag{Name: "prompt-password", Usage: "prompt for the PDF password"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return usageError("pdf " + name + " [options] <file.pdf>")
			}
			opts, err := convertOptionsFromCommand(cmd)
			if err != nil {
				return err
			}
			value, err := fn(cmd.Args().First(), opts)
			if err != nil && opts.Password == "" && isPasswordError(err) {
				opts.Password = promptPDFPassword()
				value, err = fn(cmd.Args().First(), opts)
			}
			return writeStringOutputErr(label, value, cmd.String("o"), err)
		},
	}
}

func toHTMLCommand() *cli.Command {
	return &cli.Command{
		Name:      "to-html",
		Usage:     "Convert PDF to HTML",
		ArgsUsage: "<file.pdf>",
		Category:  "Inspect and extract",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Usage: "write output to file", TakesFile: true},
			&cli.StringFlag{Name: "mode", Value: "reflowed", Usage: "conversion mode: reflowed or positioned"},
			&cli.IntFlag{Name: "page", Usage: "1-based page number"},
			&cli.StringFlag{Name: "pages", Usage: "1-based page range, e.g. 1-3,5"},
			&cli.StringFlag{Name: "password", Usage: "PDF password"},
			&cli.BoolFlag{Name: "prompt-password", Usage: "prompt for the PDF password"},
			&cli.BoolFlag{Name: "images", Value: true, Usage: "extract images"},
			&cli.BoolFlag{Name: "tables", Value: true, Usage: "detect tables"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return usageError("pdf to-html [options] <file.pdf>")
			}
			mode := cmd.String("mode")
			if mode != "reflowed" && mode != "positioned" {
				return fmt.Errorf("invalid mode: use reflowed or positioned")
			}
			opts, err := convertOptionsFromCommand(cmd)
			if err != nil {
				return err
			}
			opts.Mode = mode
			opts.ExtractImages = cmd.Bool("images")
			opts.DetectTables = cmd.Bool("tables")
			html, err := pdf.ToHTML(cmd.Args().First(), opts)
			if err != nil && opts.Password == "" && isPasswordError(err) {
				opts.Password = promptPDFPassword()
				html, err = pdf.ToHTML(cmd.Args().First(), opts)
			}
			return writeStringOutputErr("HTML", html, cmd.String("o"), err)
		},
	}
}

func toJSONCommand() *cli.Command {
	return &cli.Command{
		Name:      "to-json",
		Usage:     "Convert PDF to structured JSON",
		ArgsUsage: "<file.pdf>",
		Category:  "Inspect and extract",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Usage: "write output to file", TakesFile: true},
			&cli.IntFlag{Name: "page", Usage: "1-based page number"},
			&cli.StringFlag{Name: "pages", Usage: "1-based page range, e.g. 1-3,5"},
			&cli.StringFlag{Name: "password", Usage: "PDF password"},
			&cli.BoolFlag{Name: "prompt-password", Usage: "prompt for the PDF password"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return usageError("pdf to-json [options] <file.pdf>")
			}
			opts, err := convertOptionsFromCommand(cmd)
			if err != nil {
				return err
			}
			opts.ExtractImages = true
			opts.DetectTables = true
			data, err := pdf.ToJSON(cmd.Args().First(), opts)
			if err != nil && opts.Password == "" && isPasswordError(err) {
				opts.Password = promptPDFPassword()
				data, err = pdf.ToJSON(cmd.Args().First(), opts)
			}
			if err != nil {
				return fmt.Errorf("converting PDF to JSON: %w", err)
			}
			if output := cmd.String("o"); output != "" {
				if err := os.WriteFile(output, data, 0644); err != nil {
					return fmt.Errorf("writing %s: %w", output, err)
				}
				fmt.Printf("Wrote JSON to %s\n", output)
				return nil
			}
			os.Stdout.Write(data)
			fmt.Println()
			return nil
		},
	}
}

func extractImagesCommand() *cli.Command {
	return &cli.Command{
		Name:      "extract-images",
		Usage:     "Extract embedded images",
		ArgsUsage: "<file.pdf>",
		Category:  "Inspect and extract",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Value: ".", Usage: "output directory", TakesFile: true},
			&cli.IntFlag{Name: "page", Usage: "1-based page number"},
			&cli.StringFlag{Name: "pages", Usage: "1-based page range, e.g. 1-3,5"},
			&cli.StringFlag{Name: "password", Usage: "PDF password"},
			&cli.BoolFlag{Name: "prompt-password", Usage: "prompt for the PDF password"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return usageError("pdf extract-images -o images <file.pdf>")
			}
			opts, err := convertOptionsFromCommand(cmd)
			if err != nil {
				return err
			}
			images, err := pdf.ExtractImages(cmd.Args().First(), opts)
			if err != nil && opts.Password == "" && isPasswordError(err) {
				opts.Password = promptPDFPassword()
				images, err = pdf.ExtractImages(cmd.Args().First(), opts)
			}
			if err != nil {
				return fmt.Errorf("extracting images: %w", err)
			}
			outputDir := cmd.String("o")
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("creating %s: %w", outputDir, err)
			}
			for i, img := range images {
				path := filepath.Join(outputDir, converter.ImageFilename(img, i))
				if err := os.WriteFile(path, img.Data, 0644); err != nil {
					return fmt.Errorf("writing %s: %w", path, err)
				}
			}
			fmt.Printf("Extracted %d image(s) to %s\n", len(images), outputDir)
			return nil
		},
	}
}

func splitCommand() *cli.Command {
	return &cli.Command{
		Name:      "split",
		Usage:     "Extract pages or split from JSON config",
		ArgsUsage: "<file.pdf>",
		Category:  "Pages",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Usage: "JSON split config", TakesFile: true},
			&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Usage: "output PDF path", TakesFile: true},
			&cli.IntFlag{Name: "page", Usage: "1-based page number to extract"},
			&cli.StringFlag{Name: "pages", Usage: "1-based page range, e.g. 1-3,5"},
			&cli.StringFlag{Name: "password", Usage: "PDF password"},
			&cli.BoolFlag{Name: "prompt-password", Usage: "prompt for the PDF password"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if config := cmd.String("config"); config != "" {
				if err := pdf.SplitWithConfig(config); err != nil {
					return fmt.Errorf("splitting PDF: %w", err)
				}
				fmt.Printf("Split PDF from config %s\n", config)
				return nil
			}
			if cmd.NArg() < 1 || cmd.String("o") == "" {
				return usageError("pdf split [-config split.json] -o output.pdf [options] <file.pdf>")
			}
			pages, err := pagesFromCommand(cmd)
			if err != nil {
				return err
			}
			opts := converter.ConvertOptions{Password: passwordValue(cmd.String("password"), cmd.Bool("prompt-password"))}
			err = pdf.Split(cmd.Args().First(), cmd.String("o"), pages, opts)
			if err != nil && opts.Password == "" && isPasswordError(err) {
				opts.Password = promptPDFPassword()
				err = pdf.Split(cmd.Args().First(), cmd.String("o"), pages, opts)
			}
			if err != nil {
				return fmt.Errorf("splitting PDF: %w", err)
			}
			fmt.Printf("Wrote PDF to %s\n", cmd.String("o"))
			return nil
		},
	}
}

func deletePagesCommand() *cli.Command {
	return pageListCommand("delete-pages", "Remove selected pages", "pdf delete-pages -pages 2,4 -o output.pdf <file.pdf>", "deleting pages", func(input, output string, pages []int, opts converter.ConvertOptions) error {
		return pdf.DeletePages(input, output, pages, opts)
	})
}

func reorderCommand() *cli.Command {
	return pageListCommand("reorder", "Reorder pages", "pdf reorder -pages 3,1,2 -o output.pdf <file.pdf>", "reordering pages", func(input, output string, pages []int, opts converter.ConvertOptions) error {
		return pdf.Reorder(input, output, pages, opts)
	})
}

func pageListCommand(name, usage, usageText, actionName string, fn func(string, string, []int, converter.ConvertOptions) error) *cli.Command {
	return &cli.Command{
		Name:      name,
		Usage:     usage,
		ArgsUsage: "<file.pdf>",
		Category:  "Pages",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Usage: "output PDF path", TakesFile: true, Required: true},
			&cli.StringFlag{Name: "pages", Usage: "1-based pages, e.g. 1-3,5", Required: true},
			&cli.StringFlag{Name: "password", Usage: "PDF password"},
			&cli.BoolFlag{Name: "prompt-password", Usage: "prompt for the PDF password"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return usageError(usageText)
			}
			pages, err := pdf.ParsePageSpec(cmd.String("pages"))
			if err != nil {
				return fmt.Errorf("invalid pages: %w", err)
			}
			opts := converter.ConvertOptions{Password: passwordValue(cmd.String("password"), cmd.Bool("prompt-password"))}
			err = fn(cmd.Args().First(), cmd.String("o"), pages, opts)
			if err != nil && opts.Password == "" && isPasswordError(err) {
				opts.Password = promptPDFPassword()
				err = fn(cmd.Args().First(), cmd.String("o"), pages, opts)
			}
			if err != nil {
				return fmt.Errorf("%s: %w", actionName, err)
			}
			fmt.Printf("Wrote PDF to %s\n", cmd.String("o"))
			return nil
		},
	}
}

func rotateCommand() *cli.Command {
	return &cli.Command{
		Name:      "rotate",
		Usage:     "Rotate selected pages",
		ArgsUsage: "<file.pdf>",
		Category:  "Pages",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Usage: "output PDF path", TakesFile: true, Required: true},
			&cli.StringFlag{Name: "pages", Usage: "1-based pages to rotate, default all"},
			&cli.IntFlag{Name: "degrees", Value: 90, Usage: "rotation: 0, 90, 180, or 270"},
			&cli.StringFlag{Name: "password", Usage: "PDF password"},
			&cli.BoolFlag{Name: "prompt-password", Usage: "prompt for the PDF password"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return usageError("pdf rotate -degrees 90 -o output.pdf <file.pdf>")
			}
			var pages []int
			if spec := cmd.String("pages"); spec != "" {
				var err error
				pages, err = pdf.ParsePageSpec(spec)
				if err != nil {
					return fmt.Errorf("invalid pages: %w", err)
				}
			}
			opts := converter.ConvertOptions{Password: passwordValue(cmd.String("password"), cmd.Bool("prompt-password"))}
			err := pdf.Rotate(cmd.Args().First(), cmd.String("o"), pages, cmd.Int("degrees"), opts)
			if err != nil && opts.Password == "" && isPasswordError(err) {
				opts.Password = promptPDFPassword()
				err = pdf.Rotate(cmd.Args().First(), cmd.String("o"), pages, cmd.Int("degrees"), opts)
			}
			if err != nil {
				return fmt.Errorf("rotating PDF: %w", err)
			}
			fmt.Printf("Wrote PDF to %s\n", cmd.String("o"))
			return nil
		},
	}
}

func passwordCommand() *cli.Command {
	return &cli.Command{
		Name:     "password",
		Usage:    "Add or remove a PDF password using hidden prompts",
		Category: "Security and signing",
		Commands: []*cli.Command{
			{
				Name:      "add",
				Usage:     "Add a password",
				ArgsUsage: "<file.pdf>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Usage: "output protected PDF path", TakesFile: true, Required: true},
					&cli.StringFlag{Name: "password", Usage: "scripting escape hatch; omit for hidden prompt"},
					&cli.StringFlag{Name: "owner-password", Usage: "owner password"},
					&cli.StringFlag{Name: "input-password", Usage: "existing input PDF password"},
					&cli.BoolFlag{Name: "prompt-input-password", Usage: "prompt for the existing input PDF password"},
				},
				Action: func(_ context.Context, cmd *cli.Command) error {
					if cmd.NArg() < 1 {
						return usageError("pdf password add -o protected.pdf <file.pdf>")
					}
					userPassword := cmd.String("password")
					if userPassword == "" {
						userPassword = promptPDFPasswordLabel("New PDF password: ")
					}
					owner := cmd.String("owner-password")
					if owner == "" {
						owner = userPassword
					}
					opts := converter.ConvertOptions{Password: passwordValue(cmd.String("input-password"), cmd.Bool("prompt-input-password"))}
					err := pdf.Protect(cmd.Args().First(), cmd.String("o"), core.EncryptionConfig{
						Algorithm:     core.AES_128,
						UserPassword:  userPassword,
						OwnerPassword: owner,
						Permissions:   0xFFFFF0C4,
					}, opts)
					if err != nil && opts.Password == "" && isPasswordError(err) {
						opts.Password = promptPDFPassword()
						err = pdf.Protect(cmd.Args().First(), cmd.String("o"), core.EncryptionConfig{
							Algorithm:     core.AES_128,
							UserPassword:  userPassword,
							OwnerPassword: owner,
							Permissions:   0xFFFFF0C4,
						}, opts)
					}
					if err != nil {
						return fmt.Errorf("adding PDF password: %w", err)
					}
					fmt.Printf("Wrote password-protected PDF to %s\n", cmd.String("o"))
					return nil
				},
			},
			{
				Name:      "remove",
				Usage:     "Remove a password",
				ArgsUsage: "<file.pdf>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Usage: "output unlocked PDF path", TakesFile: true, Required: true},
					&cli.StringFlag{Name: "password", Usage: "scripting escape hatch; omit for hidden prompt"},
					&cli.BoolFlag{Name: "prompt-password", Usage: "force prompt for PDF password"},
				},
				Action: func(_ context.Context, cmd *cli.Command) error {
					if cmd.NArg() < 1 {
						return usageError("pdf password remove -o unlocked.pdf <file.pdf>")
					}
					pass := passwordValue(cmd.String("password"), cmd.Bool("prompt-password"))
					if pass == "" {
						pass = promptPDFPassword()
					}
					if err := pdf.Decrypt(cmd.Args().First(), cmd.String("o"), converter.ConvertOptions{Password: pass}); err != nil {
						return fmt.Errorf("removing PDF password: %w", err)
					}
					fmt.Printf("Wrote unlocked PDF to %s\n", cmd.String("o"))
					return nil
				},
			},
		},
	}
}

func protectCommand() *cli.Command {
	return &cli.Command{
		Name:      "protect",
		Usage:     "Legacy alias-style command for password add",
		ArgsUsage: "<file.pdf>",
		Category:  "Security and signing",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Usage: "output PDF path", TakesFile: true, Required: true},
			&cli.StringFlag{Name: "user-password", Usage: "user/open password", Required: true},
			&cli.StringFlag{Name: "owner-password", Usage: "owner password"},
			&cli.StringFlag{Name: "password", Usage: "input PDF password"},
			&cli.BoolFlag{Name: "prompt-password", Usage: "prompt for the input PDF password"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return usageError("pdf protect -user-password secret -o protected.pdf <file.pdf>")
			}
			owner := cmd.String("owner-password")
			if owner == "" {
				owner = cmd.String("user-password")
			}
			opts := converter.ConvertOptions{Password: passwordValue(cmd.String("password"), cmd.Bool("prompt-password"))}
			err := pdf.Protect(cmd.Args().First(), cmd.String("o"), core.EncryptionConfig{
				Algorithm:     core.AES_128,
				UserPassword:  cmd.String("user-password"),
				OwnerPassword: owner,
				Permissions:   0xFFFFF0C4,
			}, opts)
			if err != nil && opts.Password == "" && isPasswordError(err) {
				opts.Password = promptPDFPassword()
				err = pdf.Protect(cmd.Args().First(), cmd.String("o"), core.EncryptionConfig{
					Algorithm:     core.AES_128,
					UserPassword:  cmd.String("user-password"),
					OwnerPassword: owner,
					Permissions:   0xFFFFF0C4,
				}, opts)
			}
			if err != nil {
				return fmt.Errorf("protecting PDF: %w", err)
			}
			fmt.Printf("Wrote protected PDF to %s\n", cmd.String("o"))
			return nil
		},
	}
}

func decryptCommand() *cli.Command {
	return &cli.Command{
		Name:      "decrypt",
		Usage:     "Legacy alias-style command for password remove",
		ArgsUsage: "<file.pdf>",
		Category:  "Security and signing",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Usage: "output PDF path", TakesFile: true, Required: true},
			&cli.StringFlag{Name: "password", Usage: "PDF password"},
			&cli.BoolFlag{Name: "prompt-password", Usage: "prompt for the PDF password"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return usageError("pdf decrypt -prompt-password -o unlocked.pdf <file.pdf>")
			}
			opts := converter.ConvertOptions{Password: passwordValue(cmd.String("password"), cmd.Bool("prompt-password"))}
			err := pdf.Decrypt(cmd.Args().First(), cmd.String("o"), opts)
			if err != nil && opts.Password == "" && isPasswordError(err) {
				opts.Password = promptPDFPassword()
				err = pdf.Decrypt(cmd.Args().First(), cmd.String("o"), opts)
			}
			if err != nil {
				return fmt.Errorf("decrypting PDF: %w", err)
			}
			fmt.Printf("Wrote decrypted PDF to %s\n", cmd.String("o"))
			return nil
		},
	}
}

func signCommand() *cli.Command {
	return &cli.Command{
		Name:      "sign",
		Usage:     "Digitally sign a PDF",
		ArgsUsage: "<file.pdf>",
		Category:  "Security and signing",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Usage: "output signed PDF path", TakesFile: true, Required: true},
			&cli.StringFlag{Name: "key", Usage: "PEM private key", TakesFile: true, Required: true},
			&cli.StringFlag{Name: "cert", Usage: "PEM certificate chain", TakesFile: true, Required: true},
			&cli.StringFlag{Name: "reason", Usage: "signature reason"},
			&cli.StringFlag{Name: "location", Usage: "signature location"},
			&cli.StringFlag{Name: "contact", Usage: "signature contact"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return usageError("pdf sign -key key.pem -cert cert.pem -o signed.pdf <file.pdf>")
			}
			signer, err := loadLocalSigner(cmd.String("key"), cmd.String("cert"))
			if err != nil {
				return err
			}
			opts := sign.Options{
				Signer:      signer,
				Reason:      cmd.String("reason"),
				Location:    cmd.String("location"),
				ContactInfo: cmd.String("contact"),
			}
			if err := sign.SignFile(cmd.Args().First(), cmd.String("o"), opts); err != nil {
				return fmt.Errorf("signing PDF: %w", err)
			}
			fmt.Printf("Wrote signed PDF to %s\n", cmd.String("o"))
			return nil
		},
	}
}

func watermarkCommand() *cli.Command {
	return &cli.Command{
		Name:      "watermark",
		Usage:     "Add a text watermark",
		ArgsUsage: "<file.pdf>",
		Category:  "Modify",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Usage: "output PDF path", TakesFile: true, Required: true},
			&cli.StringFlag{Name: "text", Usage: "watermark text", Required: true},
			&cli.StringFlag{Name: "pages", Usage: "1-based pages, default all"},
			&cli.FloatFlag{Name: "size", Value: 48, Usage: "font size"},
			&cli.FloatFlag{Name: "angle", Value: 35, Usage: "angle in degrees"},
			&cli.StringFlag{Name: "password", Usage: "PDF password"},
			&cli.BoolFlag{Name: "prompt-password", Usage: "prompt for the PDF password"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return usageError("pdf watermark -text DRAFT -o output.pdf <file.pdf>")
			}
			var pages []int
			if spec := cmd.String("pages"); spec != "" {
				var err error
				pages, err = pdf.ParsePageSpec(spec)
				if err != nil {
					return fmt.Errorf("invalid pages: %w", err)
				}
			}
			err := pdf.Watermark(cmd.Args().First(), cmd.String("o"), pdf.WatermarkOptions{
				Text:     cmd.String("text"),
				FontSize: cmd.Float("size"),
				Angle:    cmd.Float("angle"),
				Pages:    pages,
				Password: passwordValue(cmd.String("password"), cmd.Bool("prompt-password")),
			})
			if err != nil {
				return fmt.Errorf("adding watermark: %w", err)
			}
			fmt.Printf("Wrote watermarked PDF to %s\n", cmd.String("o"))
			return nil
		},
	}
}

func pageNumbersCommand() *cli.Command {
	return &cli.Command{
		Name:      "page-numbers",
		Usage:     "Add page numbers",
		ArgsUsage: "<file.pdf>",
		Category:  "Modify",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Usage: "output PDF path", TakesFile: true, Required: true},
			&cli.StringFlag{Name: "format", Value: "Page %d of %d", Usage: "fmt format with page and total"},
			&cli.FloatFlag{Name: "font-size", Value: 10, Usage: "font size"},
			&cli.FloatFlag{Name: "margin", Value: 36, Usage: "bottom margin"},
			&cli.StringFlag{Name: "password", Usage: "PDF password"},
			&cli.BoolFlag{Name: "prompt-password", Usage: "prompt for the PDF password"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return usageError("pdf page-numbers -o output.pdf <file.pdf>")
			}
			err := pdf.AddPageNumbers(cmd.Args().First(), cmd.String("o"), pdf.PageNumberOptions{
				Format:   cmd.String("format"),
				FontSize: cmd.Float("font-size"),
				Margin:   cmd.Float("margin"),
				Password: passwordValue(cmd.String("password"), cmd.Bool("prompt-password")),
			})
			if err != nil {
				return fmt.Errorf("adding page numbers: %w", err)
			}
			fmt.Printf("Wrote numbered PDF to %s\n", cmd.String("o"))
			return nil
		},
	}
}

func setMetadataCommand() *cli.Command {
	return &cli.Command{
		Name:      "set-metadata",
		Usage:     "Replace document metadata",
		ArgsUsage: "<file.pdf>",
		Category:  "Modify",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "o", Aliases: []string{"output"}, Usage: "output PDF path", TakesFile: true, Required: true},
			&cli.StringFlag{Name: "title", Usage: "metadata title"},
			&cli.StringFlag{Name: "author", Usage: "metadata author"},
			&cli.StringFlag{Name: "subject", Usage: "metadata subject"},
			&cli.StringFlag{Name: "keywords", Usage: "metadata keywords"},
			&cli.StringFlag{Name: "creator", Usage: "metadata creator"},
			&cli.StringFlag{Name: "producer", Usage: "metadata producer"},
			&cli.StringFlag{Name: "password", Usage: "PDF password"},
			&cli.BoolFlag{Name: "prompt-password", Usage: "prompt for the PDF password"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return usageError("pdf set-metadata -title Title -o output.pdf <file.pdf>")
			}
			metadata := map[string]string{}
			for _, key := range []string{"title", "author", "subject", "keywords", "creator", "producer"} {
				if value := cmd.String(key); value != "" {
					metadata[strings.Title(key)] = value
				}
			}
			if len(metadata) == 0 {
				return fmt.Errorf("at least one metadata field is required")
			}
			opts := converter.ConvertOptions{Password: passwordValue(cmd.String("password"), cmd.Bool("prompt-password"))}
			err := pdf.SetMetadata(cmd.Args().First(), cmd.String("o"), metadata, opts)
			if err != nil && opts.Password == "" && isPasswordError(err) {
				opts.Password = promptPDFPassword()
				err = pdf.SetMetadata(cmd.Args().First(), cmd.String("o"), metadata, opts)
			}
			if err != nil {
				return fmt.Errorf("setting metadata: %w", err)
			}
			fmt.Printf("Wrote metadata PDF to %s\n", cmd.String("o"))
			return nil
		},
	}
}

func convertOptionsFromCommand(cmd *cli.Command) (converter.ConvertOptions, error) {
	opts := converter.ConvertOptions{Password: passwordValue(cmd.String("password"), cmd.Bool("prompt-password"))}
	if page := cmd.Int("page"); page > 0 {
		opts.Pages = []int{page - 1}
	} else if spec := cmd.String("pages"); spec != "" {
		pages, err := pdf.ParsePageSpec(spec)
		if err != nil {
			return opts, fmt.Errorf("invalid page range: %w", err)
		}
		opts.Pages = pages
	}
	return opts, nil
}

func pagesFromCommand(cmd *cli.Command) ([]int, error) {
	if page := cmd.Int("page"); page > 0 {
		return []int{page - 1}, nil
	}
	if spec := cmd.String("pages"); spec != "" {
		pages, err := pdf.ParsePageSpec(spec)
		if err != nil {
			return nil, fmt.Errorf("invalid page range: %w", err)
		}
		return pages, nil
	}
	return nil, nil
}

func printInfo(info *pdf.PDFInfo) {
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

func writeStringOutputErr(label, value, output string, err error) error {
	if err != nil {
		return fmt.Errorf("converting PDF to %s: %w", strings.ToLower(label), err)
	}
	if output != "" {
		if err := os.WriteFile(output, []byte(value), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", output, err)
		}
		fmt.Printf("Wrote %s to %s\n", label, output)
		return nil
	}
	fmt.Print(value)
	if !strings.HasSuffix(value, "\n") {
		fmt.Println()
	}
	return nil
}

func usageError(usage string) error {
	return fmt.Errorf("usage: %s", usage)
}
