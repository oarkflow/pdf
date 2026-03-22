package main

import (
	"fmt"
	"os"

	"github.com/oarkflow/pdf"
	"github.com/oarkflow/pdf/html"
)

const spotlightURL = "https://spotlight.tailwindui.com/"

func main() {
	output := "spotlight.pdf"
	if len(os.Args) > 1 {
		output = os.Args[1]
	}

	err := pdf.FromURL(spotlightURL, output, html.Options{
		BaseURL:     spotlightURL,
		MediaType:   "screen",
		UseTailwind: true,
		PageSize:    [2]float64{pdf.A3.Width, pdf.A3.Height},
		Margins:     [4]float64{24, 24, 24, 24},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating PDF: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s from %s\n", output, spotlightURL)
}
