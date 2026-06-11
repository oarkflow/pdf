package pdf

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/oarkflow/pdf/reader"
)

// ComplianceProfile identifies a PDF compliance profile.
type ComplianceProfile string

const (
	ProfileAuto   ComplianceProfile = "auto"
	ProfilePDF    ComplianceProfile = "pdf"
	ProfilePDFA1b ComplianceProfile = "pdfa-1b"
	ProfilePDFA2b ComplianceProfile = "pdfa-2b"
	ProfilePDFA4  ComplianceProfile = "pdfa-4"
	ProfilePDFUA1 ComplianceProfile = "pdfua-1"
	ProfilePDFUA2 ComplianceProfile = "pdfua-2"
	ProfilePDFX   ComplianceProfile = "pdfx-*"
	ProfilePDFE   ComplianceProfile = "pdfe-*"
	ProfilePDFVT  ComplianceProfile = "pdfvt-*"
	ProfilePAdES  ComplianceProfile = "pades-*"
)

const (
	IssueError   = "error"
	IssueWarning = "warning"
	IssueInfo    = "info"
)

// ComplianceOptions controls profile-aware PDF validation.
type ComplianceOptions struct {
	Profiles  []ComplianceProfile `json:"profiles,omitempty"`
	Password  string              `json:"password,omitempty"`
	Strict    bool                `json:"strict,omitempty"`
	External  string              `json:"external,omitempty"`
	MaxIssues int                 `json:"maxIssues,omitempty"`
}

// ComplianceReport describes a profile-aware validation result.
type ComplianceReport struct {
	Path             string              `json:"path,omitempty"`
	Valid            bool                `json:"valid"`
	Encrypted        bool                `json:"encrypted"`
	Profiles         []ComplianceProfile `json:"profiles,omitempty"`
	DetectedProfiles []ComplianceProfile `json:"detectedProfiles,omitempty"`
	Issues           []ComplianceIssue   `json:"issues,omitempty"`
	Summary          map[string]int      `json:"summary,omitempty"`
	Pages            int                 `json:"pages"`
	Metadata         map[string]string   `json:"metadata,omitempty"`
	Error            string              `json:"error,omitempty"`
}

// ComplianceIssue describes one validation finding.
type ComplianceIssue struct {
	Severity   string            `json:"severity"`
	Profile    ComplianceProfile `json:"profile,omitempty"`
	Clause     string            `json:"clause,omitempty"`
	Object     string            `json:"object,omitempty"`
	Page       int               `json:"page,omitempty"`
	Message    string            `json:"message"`
	Suggestion string            `json:"suggestion,omitempty"`
}

// ExternalValidator validates PDFs with an optional authoritative tool.
type ExternalValidator interface {
	Name() string
	Available() bool
	Validate(path string, profiles []ComplianceProfile) []ComplianceIssue
}

type complianceInspection struct {
	path       string
	data       []byte
	header     string
	reader     *reader.Reader
	resolver   *reader.Resolver
	trailer    map[string]interface{}
	catalog    map[string]interface{}
	pages      []map[string]interface{}
	metadata   map[string]string
	xmp        string
	encrypted  bool
	objectNums []int
}

// ValidateCompliance validates a PDF file against one or more compliance profiles.
func ValidateCompliance(path string, opts ComplianceOptions) ComplianceReport {
	report := ComplianceReport{Path: path, Summary: make(map[string]int)}
	if strings.TrimSpace(path) == "" {
		report.Error = "input path is empty"
		report.finish(opts.Strict)
		return report
	}
	data, err := os.ReadFile(path)
	if err != nil {
		report.Error = err.Error()
		report.finish(opts.Strict)
		return report
	}
	return validateBytes(data, path, opts)
}

// ValidateBytes validates PDF data against one or more compliance profiles.
func ValidateBytes(data []byte, opts ComplianceOptions) ComplianceReport {
	return validateBytes(data, "", opts)
}

// DetectComplianceProfiles detects compliance profiles declared by a PDF file.
func DetectComplianceProfiles(path string, password string) ([]ComplianceProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ins, issues := inspectCompliance(data, path, password)
	if len(issues) > 0 {
		for _, issue := range issues {
			if issue.Severity == IssueError {
				return nil, errors.New(issue.Message)
			}
		}
	}
	return detectProfiles(ins), nil
}

func validateBytes(data []byte, path string, opts ComplianceOptions) ComplianceReport {
	report := ComplianceReport{Path: path, Summary: make(map[string]int)}
	ins, issues := inspectCompliance(data, path, opts.Password)
	report.addIssues(issues...)
	report.Encrypted = ins.encrypted
	report.Pages = len(ins.pages)
	report.Metadata = ins.metadata
	report.DetectedProfiles = detectProfiles(ins)
	profiles := normalizeProfiles(opts.Profiles, report.DetectedProfiles)
	report.Profiles = profiles

	if len(data) == 0 {
		report.addIssue(IssueError, ProfilePDF, "", "", 0, "PDF data is empty", "provide a non-empty PDF")
		report.finish(opts.Strict)
		return report
	}
	if ins.reader != nil {
		validateCore(ins, &report)
		for _, profile := range profiles {
			switch {
			case profile == ProfilePDF:
				continue
			case isPDFA(profile):
				validatePDFA(ins, profile, &report)
			case isPDFUA(profile):
				validatePDFUA(ins, profile, &report)
			case isUnsupportedNative(profile):
				report.addIssue(IssueError, profile, "unsupported-native-profile", "", 0,
					fmt.Sprintf("%s native validation is not implemented yet", profile),
					"configure an external validator for authoritative validation")
			default:
				report.addIssue(IssueWarning, profile, "unknown-profile", "", 0,
					fmt.Sprintf("unknown compliance profile %q", profile),
					"use a known profile such as pdfa-2b, pdfa-4, pdfua-1, or pdfua-2")
			}
		}
	}

	if opts.External != "" {
		report.addIssues(runExternalValidation(data, path, opts)...)
	}
	report.limitIssues(opts.MaxIssues)
	report.finish(opts.Strict)
	return report
}

func inspectCompliance(data []byte, path, password string) (complianceInspection, []ComplianceIssue) {
	ins := complianceInspection{
		path:     path,
		data:     data,
		metadata: make(map[string]string),
	}
	if len(data) >= 8 && bytes.HasPrefix(data, []byte("%PDF-")) {
		ins.header = string(data[:8])
	}
	var issues []ComplianceIssue
	if encrypted, _ := reader.IsEncrypted(data); encrypted {
		ins.encrypted = true
	}
	r, err := reader.OpenWithPassword(data, password)
	if err != nil {
		issues = append(issues, ComplianceIssue{
			Severity:   IssueError,
			Profile:    ProfilePDF,
			Clause:     "parse",
			Message:    err.Error(),
			Suggestion: "ensure the file is a readable PDF and provide the correct password if encrypted",
		})
		return ins, issues
	}
	ins.reader = r
	ins.resolver = r.GetResolver()
	ins.trailer = r.Trailer()
	ins.catalog = r.Catalog()
	ins.pages = r.PageDictionaries()
	ins.metadata = r.Metadata()
	ins.objectNums = ins.resolver.ObjectNumbers()
	ins.xmp = extractXMP(ins)
	return ins, issues
}

func normalizeProfiles(profiles, detected []ComplianceProfile) []ComplianceProfile {
	if len(profiles) == 0 {
		profiles = []ComplianceProfile{ProfilePDF}
	}
	out := make([]ComplianceProfile, 0, len(profiles)+len(detected)+1)
	seen := map[ComplianceProfile]bool{}
	add := func(p ComplianceProfile) {
		p = normalizeProfile(p)
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	auto := false
	for _, p := range profiles {
		if normalizeProfile(p) == ProfileAuto {
			auto = true
			break
		}
	}
	add(ProfilePDF)
	if auto {
		for _, p := range detected {
			add(p)
		}
	} else {
		for _, p := range profiles {
			add(p)
		}
	}
	if len(out) == 1 && out[0] == ProfilePDF && auto {
		return out
	}
	return out
}

func normalizeProfile(profile ComplianceProfile) ComplianceProfile {
	p := ComplianceProfile(strings.ToLower(strings.TrimSpace(string(profile))))
	switch p {
	case "", ProfileAuto, ProfilePDF, ProfilePDFA1b, ProfilePDFA2b, ProfilePDFA4, ProfilePDFUA1, ProfilePDFUA2, ProfilePDFX, ProfilePDFE, ProfilePDFVT, ProfilePAdES:
		return p
	case "pdf/a-1b", "pdfa1b", "pdf/a-1-b":
		return ProfilePDFA1b
	case "pdf/a-2b", "pdfa2b", "pdf/a-2-b":
		return ProfilePDFA2b
	case "pdf/a-4", "pdfa4":
		return ProfilePDFA4
	case "pdf/ua-1", "pdfua1":
		return ProfilePDFUA1
	case "pdf/ua-2", "pdfua2":
		return ProfilePDFUA2
	}
	if strings.HasPrefix(string(p), "pdfx") || strings.HasPrefix(string(p), "pdf/x") {
		return ProfilePDFX
	}
	if strings.HasPrefix(string(p), "pdfe") || strings.HasPrefix(string(p), "pdf/e") {
		return ProfilePDFE
	}
	if strings.HasPrefix(string(p), "pdfvt") || strings.HasPrefix(string(p), "pdf/vt") {
		return ProfilePDFVT
	}
	if strings.HasPrefix(string(p), "pades") || strings.HasPrefix(string(p), "pades-") {
		return ProfilePAdES
	}
	return p
}

func detectProfiles(ins complianceInspection) []ComplianceProfile {
	var profiles []ComplianceProfile
	xmp := ins.xmp
	if xmp == "" {
		return profiles
	}
	switch textBetween(xmp, "<pdfaid:part>", "</pdfaid:part>") {
	case "1":
		if strings.EqualFold(textBetween(xmp, "<pdfaid:conformance>", "</pdfaid:conformance>"), "B") {
			profiles = append(profiles, ProfilePDFA1b)
		}
	case "2":
		profiles = append(profiles, ProfilePDFA2b)
	case "4":
		profiles = append(profiles, ProfilePDFA4)
	}
	switch textBetween(xmp, "<pdfuaid:part>", "</pdfuaid:part>") {
	case "1":
		profiles = append(profiles, ProfilePDFUA1)
	case "2":
		profiles = append(profiles, ProfilePDFUA2)
	}
	if strings.Contains(xmp, "pdfxid:") || strings.Contains(xmp, "PDF/X") {
		profiles = append(profiles, ProfilePDFX)
	}
	if strings.Contains(xmp, "PDF/E") {
		profiles = append(profiles, ProfilePDFE)
	}
	if strings.Contains(xmp, "PDF/VT") {
		profiles = append(profiles, ProfilePDFVT)
	}
	return uniqueProfiles(profiles)
}

func validateCore(ins complianceInspection, report *ComplianceReport) {
	if !strings.HasPrefix(ins.header, "%PDF-") {
		report.addIssue(IssueError, ProfilePDF, "header", "", 0, "missing or invalid PDF header", "ensure the file starts with %PDF-")
	}
	if ins.trailer == nil {
		report.addIssue(IssueError, ProfilePDF, "trailer", "", 0, "missing trailer dictionary", "write a valid PDF trailer")
	} else if _, ok := ins.trailer["/Root"]; !ok {
		report.addIssue(IssueError, ProfilePDF, "trailer-root", "", 0, "trailer is missing /Root", "set trailer /Root to the catalog")
	}
	if ins.catalog == nil {
		report.addIssue(IssueError, ProfilePDF, "catalog", "", 0, "missing document catalog", "write a valid catalog dictionary")
	} else if _, ok := ins.catalog["/Pages"]; !ok {
		report.addIssue(IssueError, ProfilePDF, "catalog-pages", "", 0, "catalog is missing /Pages", "set catalog /Pages to the page tree")
	}
	if len(ins.pages) == 0 {
		report.addIssue(IssueError, ProfilePDF, "pages", "", 0, "PDF contains no readable pages", "add at least one page")
	}
	for i := range ins.pages {
		if _, err := ins.reader.Page(i); err != nil {
			report.addIssue(IssueError, ProfilePDF, "page", "", i+1, fmt.Sprintf("page cannot be read: %v", err), "fix page tree resources and contents")
		}
	}
}

func validatePDFA(ins complianceInspection, profile ComplianceProfile, report *ComplianceReport) {
	if ins.encrypted || ins.trailer["/Encrypt"] != nil {
		report.addIssue(IssueError, profile, "encryption", "", 0, "PDF/A documents must not be encrypted", "generate an unencrypted archival PDF")
	}
	if _, ok := ins.trailer["/ID"]; !ok {
		report.addIssue(IssueError, profile, "document-id", "trailer", 0, "PDF/A trailer is missing /ID", "write a deterministic trailer /ID array")
	}
	if _, ok := ins.catalog["/Metadata"]; !ok {
		report.addIssue(IssueError, profile, "metadata", "catalog", 0, "PDF/A catalog is missing /Metadata", "embed an XMP metadata stream")
	}
	if _, ok := ins.catalog["/OutputIntents"]; !ok {
		report.addIssue(IssueError, profile, "output-intent", "catalog", 0, "PDF/A catalog is missing /OutputIntents", "embed an RGB ICC output intent")
	}
	if ins.xmp == "" {
		report.addIssue(IssueError, profile, "xmp", "metadata", 0, "XMP metadata stream is missing or unreadable", "embed valid XML XMP metadata")
	} else {
		validateXMPWellFormed(ins.xmp, profile, report)
		validatePDFAID(ins.xmp, profile, report)
		validateInfoXMPConsistency(ins, profile, report)
	}
	validateAnnotationFlags(ins, profile, report)
	if profile == ProfilePDFA1b {
		if containsKeyRecursive(ins, "/SMask") {
			report.addIssue(IssueError, profile, "smask", "", 0, "PDF/A-1 forbids image soft masks", "flatten transparent images before writing PDF/A-1")
		}
		if hasTransparentExtGState(ins) {
			report.addIssue(IssueError, profile, "transparency", "", 0, "PDF/A-1 forbids transparency graphics states", "flatten transparency or target PDF/A-2 or newer")
		}
	}
}

func validatePDFUA(ins complianceInspection, profile ComplianceProfile, report *ComplianceReport) {
	if ins.xmp == "" || !strings.Contains(ins.xmp, "<pdfuaid:part>") {
		report.addIssue(IssueError, profile, "pdfuaid", "metadata", 0, "PDF/UA identification metadata is missing", "embed pdfuaid:part in XMP metadata")
	}
	expected := "1"
	if profile == ProfilePDFUA2 {
		expected = "2"
	}
	if ins.xmp != "" && textBetween(ins.xmp, "<pdfuaid:part>", "</pdfuaid:part>") != expected {
		report.addIssue(IssueError, profile, "pdfuaid-part", "metadata", 0, "PDF/UA metadata part does not match requested profile", "write the matching pdfuaid:part value")
	}
	if lang, _ := ins.catalog["/Lang"].(string); strings.TrimSpace(lang) == "" {
		report.addIssue(IssueError, profile, "language", "catalog", 0, "PDF/UA catalog is missing /Lang", "set the document language")
	}
	if !catalogMarked(ins.catalog) {
		report.addIssue(IssueError, profile, "markinfo", "catalog", 0, "PDF/UA catalog is missing /MarkInfo /Marked true", "enable tagged PDF output")
	}
	structRoot, ok := ins.catalog["/StructTreeRoot"]
	if !ok {
		report.addIssue(IssueError, profile, "structure-tree", "catalog", 0, "PDF/UA catalog is missing /StructTreeRoot", "write a structure tree")
		return
	}
	rootObj, err := ins.resolver.ResolveReference(structRoot)
	if err != nil {
		report.addIssue(IssueError, profile, "structure-tree", "catalog", 0, fmt.Sprintf("StructTreeRoot cannot be resolved: %v", err), "write a resolvable StructTreeRoot")
		return
	}
	root, ok := rootObj.(map[string]interface{})
	if !ok {
		report.addIssue(IssueError, profile, "structure-tree", "catalog", 0, "StructTreeRoot is not a dictionary", "write a valid StructTreeRoot dictionary")
		return
	}
	if _, ok := root["/ParentTree"]; !ok {
		report.addIssue(IssueError, profile, "parent-tree", "StructTreeRoot", 0, "PDF/UA structure tree is missing /ParentTree", "write the parent tree for marked content")
	}
	if _, ok := root["/K"]; !ok {
		report.addIssue(IssueError, profile, "structure-kids", "StructTreeRoot", 0, "PDF/UA structure tree is missing /K", "write structure tree children")
	}
	validateFigureAltText(ins, profile, report)
}

func validateXMPWellFormed(xmp string, profile ComplianceProfile, report *ComplianceReport) {
	decoder := xml.NewDecoder(strings.NewReader(xmp))
	for {
		_, err := decoder.Token()
		if err == nil {
			continue
		}
		if err.Error() == "EOF" {
			return
		}
		report.addIssue(IssueError, profile, "xmp-xml", "metadata", 0, fmt.Sprintf("XMP metadata is not well-formed XML: %v", err), "write valid XML metadata")
		return
	}
}

func validatePDFAID(xmp string, profile ComplianceProfile, report *ComplianceReport) {
	expectedPart := map[ComplianceProfile]string{
		ProfilePDFA1b: "1",
		ProfilePDFA2b: "2",
		ProfilePDFA4:  "4",
	}[profile]
	part := textBetween(xmp, "<pdfaid:part>", "</pdfaid:part>")
	if part != expectedPart {
		report.addIssue(IssueError, profile, "pdfaid-part", "metadata", 0, "PDF/A metadata part does not match requested profile", "write the matching pdfaid:part value")
	}
	if profile != ProfilePDFA4 {
		if !strings.EqualFold(textBetween(xmp, "<pdfaid:conformance>", "</pdfaid:conformance>"), "B") {
			report.addIssue(IssueError, profile, "pdfaid-conformance", "metadata", 0, "PDF/A metadata conformance must be B for this profile", "write pdfaid:conformance B")
		}
	}
	if profile == ProfilePDFA4 && textBetween(xmp, "<pdfaid:rev>", "</pdfaid:rev>") == "" {
		report.addIssue(IssueWarning, profile, "pdfaid-revision", "metadata", 0, "PDF/A-4 metadata does not include pdfaid:rev", "include pdfaid:rev when targeting a specific PDF/A-4 revision")
	}
}

func validateInfoXMPConsistency(ins complianceInspection, profile ComplianceProfile, report *ComplianceReport) {
	for key, value := range ins.metadata {
		if value == "" {
			continue
		}
		switch key {
		case "Title":
			if !strings.Contains(ins.xmp, xmlEscapeLoose(value)) {
				report.addIssue(IssueError, profile, "info-xmp-title", "metadata", 0, "Info Title is not mirrored in XMP metadata", "keep document info and XMP title equivalent")
			}
		case "Author":
			if !strings.Contains(ins.xmp, xmlEscapeLoose(value)) {
				report.addIssue(IssueError, profile, "info-xmp-author", "metadata", 0, "Info Author is not mirrored in XMP metadata", "keep document info and XMP creator equivalent")
			}
		}
	}
}

func validateAnnotationFlags(ins complianceInspection, profile ComplianceProfile, report *ComplianceReport) {
	for pageIndex, page := range ins.pages {
		annots, ok := page["/Annots"]
		if !ok {
			continue
		}
		items, ok := resolveArray(ins.resolver, annots)
		if !ok {
			continue
		}
		for idx, item := range items {
			obj, err := ins.resolver.ResolveReference(item)
			if err != nil {
				report.addIssue(IssueError, profile, "annotation", "", pageIndex+1, fmt.Sprintf("annotation cannot be resolved: %v", err), "write resolvable annotation references")
				continue
			}
			dict, ok := obj.(map[string]interface{})
			if !ok {
				continue
			}
			flags, ok := intValue(dict["/F"])
			if !ok {
				report.addIssue(IssueError, profile, "annotation-flags", fmt.Sprintf("page %d annot %d", pageIndex+1, idx+1), pageIndex+1, "annotation is missing /F flags", "set /F 4 for printable visible annotations")
				continue
			}
			if flags&4 == 0 || flags&(1|2|32) != 0 {
				report.addIssue(IssueError, profile, "annotation-flags", fmt.Sprintf("page %d annot %d", pageIndex+1, idx+1), pageIndex+1, "annotation flags are not PDF/A compatible", "set Print and clear Hidden, Invisible, and NoView flags")
			}
		}
	}
}

func validateFigureAltText(ins complianceInspection, profile ComplianceProfile, report *ComplianceReport) {
	for _, num := range ins.objectNums {
		obj, err := ins.resolver.ResolveObject(num)
		if err != nil {
			continue
		}
		dict, ok := obj.(map[string]interface{})
		if !ok {
			continue
		}
		if typ, _ := dict["/Type"].(string); typ != "/StructElem" {
			continue
		}
		if s, _ := dict["/S"].(string); s == "/Figure" {
			if alt, _ := dict["/Alt"].(string); strings.TrimSpace(alt) == "" {
				report.addIssue(IssueWarning, profile, "figure-alt", fmt.Sprintf("%d 0 obj", num), 0, "Figure structure element is missing /Alt text", "provide alternate text for figures and meaningful images")
			}
		}
	}
}

func extractXMP(ins complianceInspection) string {
	metaRef, ok := ins.catalog["/Metadata"]
	if !ok || ins.resolver == nil {
		return ""
	}
	obj, err := ins.resolver.ResolveReference(metaRef)
	if err != nil {
		return ""
	}
	stream, ok := obj.(*reader.StreamObject)
	if !ok {
		return ""
	}
	data, err := ins.resolver.DecompressStream(stream.Dict, stream.Data)
	if err != nil {
		return ""
	}
	return string(data)
}

func containsKeyRecursive(ins complianceInspection, key string) bool {
	seen := map[interface{}]bool{}
	for _, num := range ins.objectNums {
		obj, err := ins.resolver.ResolveObject(num)
		if err != nil {
			continue
		}
		if containsKeyValue(ins.resolver, obj, key, seen, 0) {
			return true
		}
	}
	return false
}

func containsKeyValue(resolver *reader.Resolver, obj interface{}, key string, seen map[interface{}]bool, depth int) bool {
	if depth > 20 || obj == nil {
		return false
	}
	if ref, ok := obj.(reader.IndirectRef); ok {
		if seen[ref] {
			return false
		}
		seen[ref] = true
		resolved, err := resolver.ResolveObject(ref.ObjNum)
		if err != nil {
			return false
		}
		return containsKeyValue(resolver, resolved, key, seen, depth+1)
	}
	switch v := obj.(type) {
	case *reader.StreamObject:
		return containsKeyValue(resolver, v.Dict, key, seen, depth+1)
	case map[string]interface{}:
		if _, ok := v[key]; ok {
			return true
		}
		for _, child := range v {
			if containsKeyValue(resolver, child, key, seen, depth+1) {
				return true
			}
		}
	case []interface{}:
		for _, child := range v {
			if containsKeyValue(resolver, child, key, seen, depth+1) {
				return true
			}
		}
	}
	return false
}

func hasTransparentExtGState(ins complianceInspection) bool {
	for _, num := range ins.objectNums {
		obj, err := ins.resolver.ResolveObject(num)
		if err != nil {
			continue
		}
		if hasTransparencyDict(ins.resolver, obj, 0) {
			return true
		}
	}
	return false
}

func hasTransparencyDict(resolver *reader.Resolver, obj interface{}, depth int) bool {
	if depth > 12 {
		return false
	}
	if ref, ok := obj.(reader.IndirectRef); ok {
		resolved, err := resolver.ResolveObject(ref.ObjNum)
		if err != nil {
			return false
		}
		return hasTransparencyDict(resolver, resolved, depth+1)
	}
	switch v := obj.(type) {
	case *reader.StreamObject:
		return hasTransparencyDict(resolver, v.Dict, depth+1)
	case map[string]interface{}:
		if typ, _ := v["/Type"].(string); typ == "/ExtGState" {
			if alphaLessThanOne(v["/CA"]) || alphaLessThanOne(v["/ca"]) {
				return true
			}
		}
		for _, child := range v {
			if hasTransparencyDict(resolver, child, depth+1) {
				return true
			}
		}
	case []interface{}:
		for _, child := range v {
			if hasTransparencyDict(resolver, child, depth+1) {
				return true
			}
		}
	}
	return false
}

func alphaLessThanOne(v interface{}) bool {
	switch n := v.(type) {
	case float64:
		return n < 1
	case int64:
		return n < 1
	case int:
		return n < 1
	case string:
		f, err := strconv.ParseFloat(strings.TrimPrefix(n, "/"), 64)
		return err == nil && f < 1
	}
	return false
}

func catalogMarked(catalog map[string]interface{}) bool {
	mi, ok := catalog["/MarkInfo"]
	if !ok {
		return false
	}
	dict, ok := mi.(map[string]interface{})
	if !ok {
		return false
	}
	marked, ok := dict["/Marked"].(bool)
	return ok && marked
}

func resolveArray(resolver *reader.Resolver, value interface{}) ([]interface{}, bool) {
	resolved, err := resolver.ResolveReference(value)
	if err != nil {
		return nil, false
	}
	arr, ok := resolved.([]interface{})
	return arr, ok
}

func intValue(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	}
	return 0, false
}

func textBetween(s, start, end string) string {
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	i += len(start)
	j := strings.Index(s[i:], end)
	if j < 0 {
		return ""
	}
	return strings.TrimSpace(s[i : i+j])
}

func xmlEscapeLoose(s string) string {
	repl := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return repl.Replace(s)
}

func uniqueProfiles(profiles []ComplianceProfile) []ComplianceProfile {
	seen := map[ComplianceProfile]bool{}
	out := make([]ComplianceProfile, 0, len(profiles))
	for _, p := range profiles {
		p = normalizeProfile(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func isPDFA(profile ComplianceProfile) bool {
	return profile == ProfilePDFA1b || profile == ProfilePDFA2b || profile == ProfilePDFA4
}

func isPDFUA(profile ComplianceProfile) bool {
	return profile == ProfilePDFUA1 || profile == ProfilePDFUA2
}

func isUnsupportedNative(profile ComplianceProfile) bool {
	return profile == ProfilePDFX || profile == ProfilePDFE || profile == ProfilePDFVT || profile == ProfilePAdES
}

func (r *ComplianceReport) addIssue(severity string, profile ComplianceProfile, clause, object string, page int, message, suggestion string) {
	r.Issues = append(r.Issues, ComplianceIssue{
		Severity:   severity,
		Profile:    profile,
		Clause:     clause,
		Object:     object,
		Page:       page,
		Message:    message,
		Suggestion: suggestion,
	})
}

func (r *ComplianceReport) addIssues(issues ...ComplianceIssue) {
	r.Issues = append(r.Issues, issues...)
}

func (r *ComplianceReport) limitIssues(max int) {
	if max > 0 && len(r.Issues) > max {
		r.Issues = r.Issues[:max]
		r.addIssue(IssueWarning, ProfilePDF, "max-issues", "", 0, "additional validation issues were omitted", "increase MaxIssues to see every issue")
	}
}

func (r *ComplianceReport) finish(strict bool) {
	if r.Summary == nil {
		r.Summary = make(map[string]int)
	}
	for _, issue := range r.Issues {
		r.Summary[issue.Severity]++
	}
	r.Valid = r.Error == "" && r.Summary[IssueError] == 0 && (!strict || r.Summary[IssueWarning] == 0)
	if r.Error != "" {
		r.Summary[IssueError]++
		r.Valid = false
	}
	sort.Slice(r.Issues, func(i, j int) bool {
		if r.Issues[i].Severity != r.Issues[j].Severity {
			return severityRank(r.Issues[i].Severity) < severityRank(r.Issues[j].Severity)
		}
		if r.Issues[i].Profile != r.Issues[j].Profile {
			return r.Issues[i].Profile < r.Issues[j].Profile
		}
		return r.Issues[i].Clause < r.Issues[j].Clause
	})
}

func severityRank(s string) int {
	switch s {
	case IssueError:
		return 0
	case IssueWarning:
		return 1
	default:
		return 2
	}
}

type veraPDFValidator struct{}

func (veraPDFValidator) Name() string { return "verapdf" }

func (veraPDFValidator) Available() bool {
	_, err := exec.LookPath("verapdf")
	return err == nil
}

func (v veraPDFValidator) Validate(path string, profiles []ComplianceProfile) []ComplianceIssue {
	if !v.Available() {
		return []ComplianceIssue{{
			Severity:   IssueError,
			Profile:    ProfilePDF,
			Clause:     "external-validator",
			Message:    "external validator verapdf is not available on PATH",
			Suggestion: "install veraPDF or run native validation without --external verapdf",
		}}
	}
	args := []string{"--format", "text"}
	if len(profiles) > 0 {
		args = append(args, "--profile", string(profiles[0]))
	}
	args = append(args, path)
	out, err := exec.Command("verapdf", args...).CombinedOutput()
	if err != nil {
		return []ComplianceIssue{{
			Severity: IssueError,
			Profile:  ProfilePDF,
			Clause:   "external-validator",
			Message:  strings.TrimSpace(string(out)),
		}}
	}
	return nil
}

func runExternalValidation(data []byte, path string, opts ComplianceOptions) []ComplianceIssue {
	name := strings.ToLower(strings.TrimSpace(opts.External))
	if name == "" {
		return nil
	}
	var validator ExternalValidator
	switch name {
	case "verapdf", "vera":
		validator = veraPDFValidator{}
	default:
		return []ComplianceIssue{{
			Severity:   IssueError,
			Profile:    ProfilePDF,
			Clause:     "external-validator",
			Message:    fmt.Sprintf("unknown external validator %q", opts.External),
			Suggestion: "supported external validators: verapdf",
		}}
	}
	if path != "" {
		return validator.Validate(path, opts.Profiles)
	}
	tmp, err := os.CreateTemp("", "pdf-compliance-*.pdf")
	if err != nil {
		return []ComplianceIssue{{Severity: IssueError, Profile: ProfilePDF, Clause: "external-validator", Message: err.Error()}}
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return []ComplianceIssue{{Severity: IssueError, Profile: ProfilePDF, Clause: "external-validator", Message: err.Error()}}
	}
	if err := tmp.Close(); err != nil {
		return []ComplianceIssue{{Severity: IssueError, Profile: ProfilePDF, Clause: "external-validator", Message: err.Error()}}
	}
	return validator.Validate(filepath.Clean(tmpPath), opts.Profiles)
}
