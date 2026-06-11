package pdf

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oarkflow/pdf/core"
	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/html"
	"github.com/oarkflow/pdf/layout"
)

const complianceHTML = `<!doctype html><html><head><title>Compliance</title></head><body><h1>Compliance</h1><p>Report body</p></body></html>`

func TestValidateComplianceAutoDetectsPDFA2bPDFUA1(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteCompliantHTMLToPDF(&buf, complianceHTML, html.Options{DefaultFontFamily: "Helvetica"}); err != nil {
		t.Fatal(err)
	}

	report := ValidateBytes(buf.Bytes(), ComplianceOptions{Profiles: []ComplianceProfile{ProfileAuto}})
	if !report.Valid {
		t.Fatalf("expected valid report: %#v", report.Issues)
	}
	if !hasProfile(report.DetectedProfiles, ProfilePDFA2b) || !hasProfile(report.DetectedProfiles, ProfilePDFUA1) {
		t.Fatalf("detected profiles = %#v", report.DetectedProfiles)
	}
}

func TestDetectComplianceProfiles(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteCompliantHTMLToPDF(&buf, complianceHTML, html.Options{DefaultFontFamily: "Helvetica"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "compliant.pdf")
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	profiles, err := DetectComplianceProfiles(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if !hasProfile(profiles, ProfilePDFA2b) || !hasProfile(profiles, ProfilePDFUA1) {
		t.Fatalf("profiles = %#v", profiles)
	}
}

func TestValidateCompliancePDFA4PDFUA2(t *testing.T) {
	compiled, err := CompileCompliantHTMLWithOptions(complianceHTML, HTMLComplianceOptions{
		PDFA:     PDFA4,
		PDFUA:    PDFUA2,
		Language: "en-US",
	}, html.Options{DefaultFontFamily: "Helvetica"})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := compiled.WriteStreamingTo(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buf.String(), "%PDF-2.0") {
		t.Fatal("PDF/A-4 + PDF/UA-2 should use PDF 2.0 header")
	}
	report := ValidateBytes(buf.Bytes(), ComplianceOptions{Profiles: []ComplianceProfile{ProfileAuto}})
	if !report.Valid {
		t.Fatalf("expected valid PDF/A-4 + PDF/UA-2 report: %#v", report.Issues)
	}
	if !hasProfile(report.DetectedProfiles, ProfilePDFA4) || !hasProfile(report.DetectedProfiles, ProfilePDFUA2) {
		t.Fatalf("detected profiles = %#v", report.DetectedProfiles)
	}
}

func TestValidateComplianceLeanPassesPDFOnly(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteLeanHTMLToPDF(&buf, complianceHTML, html.Options{DefaultFontFamily: "Helvetica"}); err != nil {
		t.Fatal(err)
	}
	report := ValidateBytes(buf.Bytes(), ComplianceOptions{Profiles: []ComplianceProfile{ProfileAuto}})
	if !report.Valid {
		t.Fatalf("lean PDF should pass core validation: %#v", report.Issues)
	}
	if len(report.DetectedProfiles) != 0 {
		t.Fatalf("lean PDF should not claim compliance profiles: %#v", report.DetectedProfiles)
	}
}

func TestValidateComplianceEncryptedFailsPDFA(t *testing.T) {
	doc, _ := document.NewDocument(document.A4)
	doc.SetEncryption(core.EncryptionConfig{
		Algorithm:     core.RC4_128,
		OwnerPassword: "owner",
		UserPassword:  "user",
		Permissions:   0xFFFFF0C4,
	})
	page := doc.NewPage()
	page.Contents = []byte("BT ET")
	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	report := ValidateBytes(buf.Bytes(), ComplianceOptions{
		Profiles: []ComplianceProfile{ProfilePDFA2b},
		Password: "user",
	})
	if report.Valid || !hasClause(report.Issues, "encryption") {
		t.Fatalf("expected PDF/A encryption failure: %#v", report.Issues)
	}
}

func TestValidateComplianceAnnotationMissingPrintFlagFailsPDFA(t *testing.T) {
	doc, _ := document.NewDocument(document.A4)
	page := doc.NewPage()
	page.Contents = []byte("BT ET")
	page.Annotations = []layout.LinkAnnotation{{X1: 10, Y1: 10, X2: 20, Y2: 20, URI: "https://example.com"}}
	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	mutated := bytes.Replace(buf.Bytes(), []byte("/F 4"), nil, 1)
	report := ValidateBytes(mutated, ComplianceOptions{Profiles: []ComplianceProfile{ProfilePDFA2b}})
	if report.Valid || !hasClause(report.Issues, "annotation-flags") {
		t.Fatalf("expected annotation flag failure: %#v", report.Issues)
	}
}

func TestValidateCompliancePDFA1SoftMaskFails(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteCompliantHTMLToPDF(&buf, complianceHTML, html.Options{DefaultFontFamily: "Helvetica"}); err != nil {
		t.Fatal(err)
	}
	mutated := bytes.Replace(buf.Bytes(), []byte("/Subtype /XML"), []byte("/Subtype /XML /SMask 1 0 R"), 1)
	report := ValidateBytes(mutated, ComplianceOptions{Profiles: []ComplianceProfile{ProfilePDFA1b}})
	if report.Valid || !hasClause(report.Issues, "smask") {
		t.Fatalf("expected SMask failure: %#v", report.Issues)
	}
}

func TestValidateComplianceUnsupportedNativeProfileFailsExplicitly(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteLeanHTMLToPDF(&buf, complianceHTML, html.Options{DefaultFontFamily: "Helvetica"}); err != nil {
		t.Fatal(err)
	}
	report := ValidateBytes(buf.Bytes(), ComplianceOptions{Profiles: []ComplianceProfile{ProfilePDFX}})
	if report.Valid || !hasClause(report.Issues, "unsupported-native-profile") {
		t.Fatalf("expected unsupported profile failure: %#v", report.Issues)
	}
}

func TestValidateComplianceMissingExternalValidator(t *testing.T) {
	if _, err := exec.LookPath("verapdf"); err == nil {
		t.Skip("verapdf is installed")
	}
	var buf bytes.Buffer
	if err := WriteLeanHTMLToPDF(&buf, complianceHTML, html.Options{DefaultFontFamily: "Helvetica"}); err != nil {
		t.Fatal(err)
	}
	report := ValidateBytes(buf.Bytes(), ComplianceOptions{Profiles: []ComplianceProfile{ProfilePDF}, External: "verapdf"})
	if report.Valid || !hasClause(report.Issues, "external-validator") {
		t.Fatalf("expected missing external validator failure: %#v", report.Issues)
	}
}

func hasProfile(profiles []ComplianceProfile, want ComplianceProfile) bool {
	for _, profile := range profiles {
		if profile == want {
			return true
		}
	}
	return false
}

func hasClause(issues []ComplianceIssue, want string) bool {
	for _, issue := range issues {
		if issue.Clause == want {
			return true
		}
	}
	return false
}
