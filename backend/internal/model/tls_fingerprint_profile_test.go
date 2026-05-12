package model

import "testing"

func TestTLSFingerprintProfileValidate(t *testing.T) {
	profile := &TLSFingerprintProfile{Name: "claude_cli_v2"}
	if err := profile.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}

	empty := &TLSFingerprintProfile{}
	err := empty.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want validation error")
	}
	if err.Error() != "name: name is required" {
		t.Fatalf("Validate() error = %q, want %q", err.Error(), "name: name is required")
	}
}

func TestTLSFingerprintProfileToTLSProfile(t *testing.T) {
	profile := &TLSFingerprintProfile{
		Name:                "custom",
		EnableGREASE:        true,
		CipherSuites:        []uint16{0x1301, 0x1302},
		Curves:              []uint16{23, 24},
		PointFormats:        []uint16{0},
		SignatureAlgorithms: []uint16{0x0403, 0x0804},
		ALPNProtocols:       []string{"h2", "http/1.1"},
		SupportedVersions:   []uint16{0x0304, 0x0303},
		KeyShareGroups:      []uint16{29},
		PSKModes:            []uint16{1},
		Extensions:          []uint16{0, 43, 45},
	}

	got := profile.ToTLSProfile()
	if got == nil {
		t.Fatal("ToTLSProfile() = nil, want profile")
	}
	if got.Name != profile.Name {
		t.Fatalf("Name = %q, want %q", got.Name, profile.Name)
	}
	if got.EnableGREASE != profile.EnableGREASE {
		t.Fatalf("EnableGREASE = %v, want %v", got.EnableGREASE, profile.EnableGREASE)
	}
	assertUint16SliceEqual(t, "CipherSuites", got.CipherSuites, profile.CipherSuites)
	assertUint16SliceEqual(t, "Curves", got.Curves, profile.Curves)
	assertUint16SliceEqual(t, "PointFormats", got.PointFormats, profile.PointFormats)
	assertUint16SliceEqual(t, "SignatureAlgorithms", got.SignatureAlgorithms, profile.SignatureAlgorithms)
	assertStringSliceEqual(t, "ALPNProtocols", got.ALPNProtocols, profile.ALPNProtocols)
	assertUint16SliceEqual(t, "SupportedVersions", got.SupportedVersions, profile.SupportedVersions)
	assertUint16SliceEqual(t, "KeyShareGroups", got.KeyShareGroups, profile.KeyShareGroups)
	assertUint16SliceEqual(t, "PSKModes", got.PSKModes, profile.PSKModes)
	assertUint16SliceEqual(t, "Extensions", got.Extensions, profile.Extensions)
}

func assertUint16SliceEqual(t *testing.T, field string, got []uint16, want []uint16) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d", field, len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("%s[%d] = %d, want %d", field, index, got[index], want[index])
		}
	}
}

func assertStringSliceEqual(t *testing.T, field string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d", field, len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("%s[%d] = %q, want %q", field, index, got[index], want[index])
		}
	}
}
