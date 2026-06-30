package pdf

// ToolStatus describes whether a PDF tool family is implemented today.
type ToolStatus string

const (
	ToolAvailable ToolStatus = "available"
)

// ToolCapability describes one user-facing PDF tool capability.
type ToolCapability struct {
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Category    string     `json:"category"`
	Status      ToolStatus `json:"status"`
	Description string     `json:"description"`
}

// ToolCatalog returns the built-in PDF tool families. It is intended for CLIs,
// UIs, and API discovery.
func ToolCatalog() []ToolCapability {
	return []ToolCapability{
		{Key: "template-json", Name: "JSON template filling", Category: "Template tools", Status: ToolAvailable, Description: "Render {{ placeholders }} in HTML templates from JSON data and generate filled PDFs."},
		{Key: "acroform-create", Name: "AcroForm creation", Category: "Form tools", Status: ToolAvailable, Description: "Create text, checkbox, radio, dropdown, combo, and signature fields."},
		{Key: "acroform-fill", Name: "AcroForm filling", Category: "Form tools", Status: ToolAvailable, Description: "Fill existing interactive PDF form fields from JSON or key-value data into flattened output PDFs."},
		{Key: "digital-signature", Name: "Digital signatures", Category: "Signature tools", Status: ToolAvailable, Description: "Apply certificate-backed PDF signatures with reason, location, and contact metadata."},
		{Key: "signature-image", Name: "Signature image placement", Category: "Signature tools", Status: ToolAvailable, Description: "Place uploaded signature images on selected PDF pages."},
		{Key: "conversion", Name: "Conversion", Category: "Conversion tools", Status: ToolAvailable, Description: "Convert Markdown, HTML, and images to PDF, and convert PDFs to text, HTML, Markdown, JSON, and extracted images."},
		{Key: "scanner", Name: "Scanner workflow", Category: "Scanner tools", Status: ToolAvailable, Description: "Convert scanned image batches into paginated PDFs."},
		{Key: "compression", Name: "Compression", Category: "Compression tools", Status: ToolAvailable, Description: "Rewrite PDFs through compressed page streams and normalized copied resources."},
		{Key: "security", Name: "Security", Category: "Security tools", Status: ToolAvailable, Description: "Protect and decrypt PDFs with password-based encryption."},
		{Key: "redaction", Name: "Redaction", Category: "Redaction tools", Status: ToolAvailable, Description: "Remove matching literal text from content streams and cover configured regions."},
		{Key: "organization", Name: "Page organization", Category: "Organization tools", Status: ToolAvailable, Description: "Merge, split, delete, reorder, rotate, watermark, number pages, and update metadata."},
		{Key: "annotation-review", Name: "Annotation and review", Category: "Annotation/review tools", Status: ToolAvailable, Description: "Read annotations, links, widget metadata, outlines, and review-related document information."},
		{Key: "comparison", Name: "PDF comparison", Category: "Comparison tools", Status: ToolAvailable, Description: "Compare page counts, extracted text, and metadata."},
		{Key: "translation", Name: "Translation", Category: "Translation tools", Status: ToolAvailable, Description: "Apply dictionary-based text replacements inside literal PDF text streams."},
		{Key: "validation", Name: "Validation", Category: "Validation tools", Status: ToolAvailable, Description: "Validate PDF structure and compliance profiles including PDF/A, PDF/UA, PDF/X, PDF/E, PDF/VT, and PAdES."},
		{Key: "print-prep", Name: "Print preparation", Category: "Print preparation tools", Status: ToolAvailable, Description: "Normalize PDFs with metadata and optional page numbering for print handoff."},
		{Key: "archive", Name: "Archive", Category: "Archive tools", Status: ToolAvailable, Description: "Write archive copies and run PDF/A-oriented validation."},
		{Key: "graph", Name: "Related PDF graph", Category: "Related PDF graph tools", Status: ToolAvailable, Description: "Build relationship graphs across PDFs using links and metadata."},
	}
}
