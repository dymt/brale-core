package onboarding

import "testing"

func TestParseHostOwnership(t *testing.T) {
	tests := []struct {
		name    string
		uidRaw  string
		gidRaw  string
		wantUID int
		wantGID int
		wantOK  bool
		wantErr bool
	}{
		{name: "unset", wantOK: false},
		{name: "valid", uidRaw: "1000", gidRaw: "1001", wantUID: 1000, wantGID: 1001, wantOK: true},
		{name: "missing gid", uidRaw: "1000", wantErr: true},
		{name: "invalid uid", uidRaw: "abc", gidRaw: "1000", wantErr: true},
		{name: "negative gid", uidRaw: "1000", gidRaw: "-1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUID, gotGID, gotOK, err := parseHostOwnership(tt.uidRaw, tt.gidRaw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseHostOwnership() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotUID != tt.wantUID || gotGID != tt.wantGID || gotOK != tt.wantOK {
				t.Fatalf("parseHostOwnership() = (%d, %d, %v), want (%d, %d, %v)", gotUID, gotGID, gotOK, tt.wantUID, tt.wantGID, tt.wantOK)
			}
		})
	}
}
