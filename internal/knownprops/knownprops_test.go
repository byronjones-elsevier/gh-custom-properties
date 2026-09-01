package knownprops

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		kind    Kind
		value   string
		wantErr bool
	}{
		{name: "alphanumeric ok", kind: Alphanumeric, value: "ABC123"},
		{name: "alphanumeric rejects space", kind: Alphanumeric, value: "ABC 123", wantErr: true},
		{name: "alphanumeric rejects hyphen", kind: Alphanumeric, value: "ABC-123", wantErr: true},
		{name: "alphanumeric rejects empty", kind: Alphanumeric, value: "", wantErr: true},
		{name: "email ok", kind: Email, value: "b.jones1@elsevier.com"},
		{name: "email rejects missing @", kind: Email, value: "not-an-email", wantErr: true},
		{name: "date ok", kind: Date, value: "2026-09-01"},
		{name: "date rejects wrong format", kind: Date, value: "09/01/2026", wantErr: true},
		{name: "date rejects invalid calendar date", kind: Date, value: "2026-13-40", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.kind, tt.value)
			if tt.wantErr && err == nil {
				t.Errorf("Validate(%v, %q) = nil, want an error", tt.kind, tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate(%v, %q) = %v, want nil", tt.kind, tt.value, err)
			}
		})
	}
}

func TestValidators_CoversElsevierStandardStringProperties(t *testing.T) {
	for _, name := range []string{"CostCode", "SystemID", "SystemName", "owner", "MigrationReadyDate"} {
		if _, ok := Validators[name]; !ok {
			t.Errorf("Validators is missing an entry for %q", name)
		}
	}
}
