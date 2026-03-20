package forms

// FieldType enumerates the supported AcroForm field types.
type FieldType int

const (
	FieldText FieldType = iota
	FieldCheckbox
	FieldRadio
	FieldDropdown
	FieldComboBox
	FieldSignature
)

// Field represents a single interactive form field.
type Field struct {
	Name     string
	Type     FieldType
	Rect     [4]float64 // x1, y1, x2, y2
	Value    string
	Default  string
	Options  []string // for dropdown/radio
	Flags    uint32
	FontName string
	FontSize float64
	Page     int
}

// TextFieldOpts configures a text field.
type TextFieldOpts struct {
	Rect      [4]float64
	Default   string
	MaxLen    int
	Multiline bool
	Password  bool
	ReadOnly  bool
	FontSize  float64
	Page      int
}

// CheckboxOpts configures a checkbox field.
type CheckboxOpts struct {
	Rect     [4]float64
	Checked  bool
	ReadOnly bool
	Page     int
}

// RadioGroupOpts configures a radio button group.
type RadioGroupOpts struct {
	Rect     [4]float64
	Options  []string
	Default  string
	ReadOnly bool
	Page     int
}

// DropdownOpts configures a dropdown (choice) field.
type DropdownOpts struct {
	Rect     [4]float64
	Options  []string
	Default  string
	Editable bool
	ReadOnly bool
	FontSize float64
	Page     int
}

// SignatureFieldOpts configures a digital signature field.
type SignatureFieldOpts struct {
	Rect [4]float64
	Page int
}
