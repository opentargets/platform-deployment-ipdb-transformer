package cmd

import (
	"net/netip"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFromFileProducesExpectedSchema(t *testing.T) {
	input := writeInputFile(t, sampleJSONL)
	output := filepath.Join(t.TempDir(), "output.mmdb")

	if _, err := fromFile(input, output); err != nil {
		t.Fatalf("fromFile returned an error: %v", err)
	}

	reader, records := readDatabase(t, output)
	if reader.Metadata.DatabaseType != "GeoIP2-Country" {
		t.Errorf("database type = %q, want %q", reader.Metadata.DatabaseType, "GeoIP2-Country")
	}
	if reader.Metadata.IPVersion != 6 {
		t.Errorf("IP version = %d, want 6", reader.Metadata.IPVersion)
	}
	if reader.Metadata.Description["en"] != "Opentargets database of IP data" {
		t.Errorf("description = %q, want %q", reader.Metadata.Description["en"], "Opentargets database of IP data")
	}

	expected := map[string]geoIP2Country{
		"zzzz1.1.1.0/24": expectedCountry("OC", "Oceania", "AU", "Australia"),
		"8.8.8.8/32":     expectedCountry("NA", "North America", "US", "United States"),
		"2600::/16":      expectedCountry("AS", "Asia", "JP", "Japan"),
	}
	if !reflect.DeepEqual(records, expected) {
		t.Errorf("records = %#v, want %#v", records, expected)
	}
}

func TestFromFileExcludesASNFields(t *testing.T) {
	input := writeInputFile(t, sampleJSONL)
	output := filepath.Join(t.TempDir(), "output.mmdb")

	if _, err := fromFile(input, output); err != nil {
		t.Fatalf("fromFile returned an error: %v", err)
	}

	reader, _ := readDatabase(t, output)
	result := reader.Lookup(netip.MustParseAddr("1.1.1.1"))
	if !result.Found() {
		t.Fatal("expected network to be found")
	}

	var raw map[string]any
	if err := result.Decode(&raw); err != nil {
		t.Fatalf("decode raw record: %v", err)
	}
	if len(raw) != 2 {
		t.Errorf("record contains %d fields, want 2: %#v", len(raw), raw)
	}
	for _, field := range []string{"asn", "as_name", "as_domain"} {
		if _, ok := raw[field]; ok {
			t.Errorf("record contains unexpected field %q", field)
		}
	}
}

func TestFromFileAllowsMissingOptionalFields(t *testing.T) {
	input := writeInputFile(t, `{"network":"93.184.216.0/24"}`+"\n")
	output := filepath.Join(t.TempDir(), "output.mmdb")

	if _, err := fromFile(input, output); err != nil {
		t.Fatalf("fromFile returned an error: %v", err)
	}

	_, records := readDatabase(t, output)
	expected := map[string]geoIP2Country{
		"93.184.216.0/24": expectedCountry("", "", "", ""),
	}
	if !reflect.DeepEqual(records, expected) {
		t.Errorf("records = %#v, want %#v", records, expected)
	}
}

func TestFromFileReturnsErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		output  func(string) string
	}{
		{
			name:    "malformed JSON",
			content: "not json\n",
			output:  func(dir string) string { return filepath.Join(dir, "output.mmdb") },
		},
		{
			name:    "invalid network",
			content: `{"network":"not-an-ip"}` + "\n",
			output:  func(dir string) string { return filepath.Join(dir, "output.mmdb") },
		},
		{
			name:    "empty network",
			content: `{"network":""}` + "\n",
			output:  func(dir string) string { return filepath.Join(dir, "output.mmdb") },
		},
		{
			name:    "bare IPv6 address",
			content: `{"network":"2001:db8::"}` + "\n",
			output:  func(dir string) string { return filepath.Join(dir, "output.mmdb") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := writeInputFile(t, tt.content)
			output := tt.output(t.TempDir())

			if _, err := fromFile(input, output); err == nil {
				t.Fatal("fromFile returned no error")
			}
		})
	}
}

func TestFromFileReturnsErrorForMissingInput(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "output.mmdb")
	if _, err := fromFile(filepath.Join(t.TempDir(), "missing.jsonl"), output); err == nil {
		t.Fatal("fromFile returned no error")
	}
}

func TestFromFileReturnsErrorForInvalidOutput(t *testing.T) {
	t.Parallel()

	input := writeInputFile(t, sampleJSONL)
	output := filepath.Join(t.TempDir(), "missing", "output.mmdb")

	if _, err := fromFile(input, output); err == nil {
		t.Fatal("fromFile returned no error")
	}
}

func TestFromFileAcceptsEmptyInput(t *testing.T) {
	t.Parallel()

	input := writeInputFile(t, "")
	output := filepath.Join(t.TempDir(), "output.mmdb")

	if _, err := fromFile(input, output); err != nil {
		t.Fatalf("fromFile returned an error: %v", err)
	}

	reader, records := readDatabase(t, output)
	if reader.Metadata.DatabaseType != "GeoIP2-Country" {
		t.Errorf("database type = %q, want %q", reader.Metadata.DatabaseType, "GeoIP2-Country")
	}
	if len(records) != 0 {
		t.Errorf("records = %#v, want empty", records)
	}
}

func TestFromFileLastDuplicateNetworkWins(t *testing.T) {
	t.Parallel()

	input := writeInputFile(t, `{"network":"93.184.216.0/24","country_code":"US","country":"United States","continent_code":"NA","continent":"North America"}`+"\n"+`{"network":"93.184.216.0/24","country_code":"CA","country":"Canada","continent_code":"NA","continent":"North America"}`+"\n")
	output := filepath.Join(t.TempDir(), "output.mmdb")

	if _, err := fromFile(input, output); err != nil {
		t.Fatalf("fromFile returned an error: %v", err)
	}

	_, records := readDatabase(t, output)
	expected := map[string]geoIP2Country{
		"93.184.216.0/24": expectedCountry("NA", "North America", "CA", "Canada"),
	}
	if !reflect.DeepEqual(records, expected) {
		t.Errorf("records = %#v, want %#v", records, expected)
	}
}
