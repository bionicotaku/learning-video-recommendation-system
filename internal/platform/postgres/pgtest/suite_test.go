package pgtest

import "testing"

func TestValidateIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "lowercase", value: "pgtest_db", wantErr: false},
		{name: "underscore prefix", value: "_pgtest_db", wantErr: false},
		{name: "contains digit", value: "pgtest_db_1", wantErr: false},
		{name: "empty", value: "", wantErr: true},
		{name: "uppercase", value: "Pgtest", wantErr: true},
		{name: "digit prefix", value: "1_pgtest", wantErr: true},
		{name: "dash", value: "pgtest-db", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIdentifier(tt.value, "database name")
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateIdentifier() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
