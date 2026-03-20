package forms

import "github.com/oarkflow/pdf/core"

// AcroForm represents an interactive form (AcroForm) structure.
type AcroForm struct {
	Fields []*Field
}

// New creates a new empty AcroForm.
func New() *AcroForm {
	return &AcroForm{}
}

// AddTextField adds a text input field to the form.
func (af *AcroForm) AddTextField(name string, opts TextFieldOpts) *Field {
	var flags uint32
	if opts.ReadOnly {
		flags |= 1 // Ff bit 1
	}
	if opts.Multiline {
		flags |= 1 << 12 // bit 13
	}
	if opts.Password {
		flags |= 1 << 13 // bit 14
	}
	if opts.MaxLen > 0 {
		flags |= 1 << 24 // bit 25 (MaxLen presence indicated separately, but set flag)
	}
	f := &Field{
		Name:     name,
		Type:     FieldText,
		Rect:     opts.Rect,
		Default:  opts.Default,
		Value:    opts.Default,
		Flags:    flags,
		FontSize: opts.FontSize,
		Page:     opts.Page,
	}
	af.Fields = append(af.Fields, f)
	return f
}

// AddCheckbox adds a checkbox field to the form.
func (af *AcroForm) AddCheckbox(name string, opts CheckboxOpts) *Field {
	var flags uint32
	if opts.ReadOnly {
		flags |= 1
	}
	val := "Off"
	if opts.Checked {
		val = "Yes"
	}
	f := &Field{
		Name:  name,
		Type:  FieldCheckbox,
		Rect:  opts.Rect,
		Value: val,
		Flags: flags,
		Page:  opts.Page,
	}
	af.Fields = append(af.Fields, f)
	return f
}

// AddRadioGroup adds a radio button group to the form.
func (af *AcroForm) AddRadioGroup(name string, opts RadioGroupOpts) *Field {
	var flags uint32
	if opts.ReadOnly {
		flags |= 1
	}
	flags |= 1 << 14 // NoToggleToOff
	flags |= 1 << 15 // Radio
	f := &Field{
		Name:    name,
		Type:    FieldRadio,
		Rect:    opts.Rect,
		Options: opts.Options,
		Default: opts.Default,
		Value:   opts.Default,
		Flags:   flags,
		Page:    opts.Page,
	}
	af.Fields = append(af.Fields, f)
	return f
}

// AddDropdown adds a dropdown (choice) field to the form.
func (af *AcroForm) AddDropdown(name string, opts DropdownOpts) *Field {
	var flags uint32
	if opts.ReadOnly {
		flags |= 1
	}
	if opts.Editable {
		flags |= 1 << 17 // Edit bit
	}
	f := &Field{
		Name:     name,
		Type:     FieldDropdown,
		Rect:     opts.Rect,
		Options:  opts.Options,
		Default:  opts.Default,
		Value:    opts.Default,
		Flags:    flags,
		FontSize: opts.FontSize,
		Page:     opts.Page,
	}
	af.Fields = append(af.Fields, f)
	return f
}

// AddSignatureField adds a digital signature field to the form.
func (af *AcroForm) AddSignatureField(name string, opts SignatureFieldOpts) *Field {
	f := &Field{
		Name: name,
		Type: FieldSignature,
		Rect: opts.Rect,
		Page: opts.Page,
	}
	af.Fields = append(af.Fields, f)
	return f
}

// BuildObjects creates the PDF indirect objects representing the AcroForm and
// its fields. The nextObjNum function should return successive unused object
// numbers each time it is called.
func (af *AcroForm) BuildObjects(nextObjNum func() int) []core.PdfIndirectObject {
	var objects []core.PdfIndirectObject
	var fieldRefs core.PdfArray

	for _, field := range af.Fields {
		objNum := nextObjNum()

		dict := core.NewDictionary()
		dict.Set("Type", core.PdfName("Annot"))
		dict.Set("Subtype", core.PdfName("Widget"))
		dict.Set("T", core.PdfString(field.Name))
		dict.Set("Rect", core.PdfArray{
			core.PdfNumber(field.Rect[0]),
			core.PdfNumber(field.Rect[1]),
			core.PdfNumber(field.Rect[2]),
			core.PdfNumber(field.Rect[3]),
		})

		if field.Flags != 0 {
			dict.Set("Ff", core.PdfInteger(int64(field.Flags)))
		}

		switch field.Type {
		case FieldText:
			dict.Set("FT", core.PdfName("Tx"))
			if field.Value != "" {
				dict.Set("V", core.PdfString(field.Value))
			}
			if field.Default != "" {
				dict.Set("DV", core.PdfString(field.Default))
			}
		case FieldCheckbox:
			dict.Set("FT", core.PdfName("Btn"))
			if field.Value == "Yes" {
				dict.Set("V", core.PdfName("Yes"))
				dict.Set("AS", core.PdfName("Yes"))
			} else {
				dict.Set("V", core.PdfName("Off"))
				dict.Set("AS", core.PdfName("Off"))
			}
		case FieldRadio:
			dict.Set("FT", core.PdfName("Btn"))
			if field.Value != "" {
				dict.Set("V", core.PdfName(field.Value))
			}
		case FieldDropdown, FieldComboBox:
			dict.Set("FT", core.PdfName("Ch"))
			if len(field.Options) > 0 {
				opts := make(core.PdfArray, len(field.Options))
				for i, o := range field.Options {
					opts[i] = core.PdfString(o)
				}
				dict.Set("Opt", opts)
			}
			if field.Value != "" {
				dict.Set("V", core.PdfString(field.Value))
			}
		case FieldSignature:
			dict.Set("FT", core.PdfName("Sig"))
		}

		// Build and attach the appearance stream.
		ap := BuildAppearance(field)
		if ap != nil {
			apObjNum := nextObjNum()
			apObj := core.PdfIndirectObject{
				Reference: core.PdfIndirectReference{ObjectNumber: apObjNum},
				Object:    ap,
			}
			objects = append(objects, apObj)

			apDict := core.NewDictionary()
			apDict.Set("N", core.PdfIndirectReference{ObjectNumber: apObjNum})
			dict.Set("AP", apDict)
		}

		ref := core.PdfIndirectReference{ObjectNumber: objNum}
		obj := core.PdfIndirectObject{
			Reference: ref,
			Object:    dict,
		}
		objects = append(objects, obj)
		fieldRefs = append(fieldRefs, ref)
	}

	// Create the AcroForm dictionary object.
	acroDict := core.NewDictionary()
	acroDict.Set("Fields", fieldRefs)

	acroObjNum := nextObjNum()
	acroObj := core.PdfIndirectObject{
		Reference: core.PdfIndirectReference{ObjectNumber: acroObjNum},
		Object:    acroDict,
	}
	objects = append(objects, acroObj)

	return objects
}
