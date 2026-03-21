package forms

import "testing"

func TestNewAcroForm(t *testing.T) {
	af := New()
	if af == nil {
		t.Fatal("New returned nil")
	}
	if len(af.Fields) != 0 {
		t.Error("new form should have no fields")
	}
}

func TestAddTextField(t *testing.T) {
	af := New()
	f := af.AddTextField("name", TextFieldOpts{
		Rect:    [4]float64{0, 0, 200, 30},
		Default: "John",
	})
	if f.Name != "name" {
		t.Errorf("Name = %q", f.Name)
	}
	if f.Type != FieldText {
		t.Errorf("Type = %d", f.Type)
	}
	if f.Value != "John" {
		t.Errorf("Value = %q", f.Value)
	}
	if len(af.Fields) != 1 {
		t.Errorf("Fields count = %d", len(af.Fields))
	}
}

func TestAddTextFieldFlags(t *testing.T) {
	af := New()
	f := af.AddTextField("pw", TextFieldOpts{
		Rect:      [4]float64{0, 0, 100, 20},
		Password:  true,
		Multiline: true,
		ReadOnly:  true,
	})
	if f.Flags&1 == 0 {
		t.Error("ReadOnly flag not set")
	}
	if f.Flags&(1<<12) == 0 {
		t.Error("Multiline flag not set")
	}
	if f.Flags&(1<<13) == 0 {
		t.Error("Password flag not set")
	}
}

func TestAddCheckbox(t *testing.T) {
	af := New()
	f := af.AddCheckbox("agree", CheckboxOpts{Checked: true})
	if f.Type != FieldCheckbox {
		t.Error("wrong type")
	}
	if f.Value != "Yes" {
		t.Errorf("Value = %q, want Yes", f.Value)
	}
}

func TestAddCheckboxUnchecked(t *testing.T) {
	af := New()
	f := af.AddCheckbox("agree", CheckboxOpts{Checked: false})
	if f.Value != "Off" {
		t.Errorf("Value = %q, want Off", f.Value)
	}
}

func TestAddRadioGroup(t *testing.T) {
	af := New()
	f := af.AddRadioGroup("choice", RadioGroupOpts{
		Options: []string{"A", "B", "C"},
		Default: "B",
	})
	if f.Type != FieldRadio {
		t.Error("wrong type")
	}
	if f.Value != "B" {
		t.Errorf("Value = %q", f.Value)
	}
	if len(f.Options) != 3 {
		t.Errorf("Options count = %d", len(f.Options))
	}
}

func TestAddDropdown(t *testing.T) {
	af := New()
	f := af.AddDropdown("country", DropdownOpts{
		Options:  []string{"US", "UK", "CA"},
		Default:  "US",
		Editable: true,
	})
	if f.Type != FieldDropdown {
		t.Error("wrong type")
	}
	if f.Flags&(1<<17) == 0 {
		t.Error("Editable flag not set")
	}
}

func TestAddSignatureField(t *testing.T) {
	af := New()
	f := af.AddSignatureField("sig", SignatureFieldOpts{
		Rect: [4]float64{100, 100, 300, 150},
		Page: 1,
	})
	if f.Type != FieldSignature {
		t.Error("wrong type")
	}
	if f.Page != 1 {
		t.Errorf("Page = %d", f.Page)
	}
}

func TestBuildObjects(t *testing.T) {
	af := New()
	af.AddTextField("f1", TextFieldOpts{Rect: [4]float64{0, 0, 100, 20}})
	af.AddCheckbox("f2", CheckboxOpts{Checked: true})
	nextObj := 1
	objects := af.BuildObjects(func() int {
		n := nextObj
		nextObj++
		return n
	})
	if len(objects) == 0 {
		t.Error("no objects created")
	}
}

func TestFieldTypes(t *testing.T) {
	if FieldText != 0 || FieldCheckbox != 1 || FieldRadio != 2 || FieldDropdown != 3 {
		t.Error("FieldType constants changed")
	}
}
