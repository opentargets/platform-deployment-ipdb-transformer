package cmd

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/maxmind/mmdbwriter/mmdbtype"
)

func TestTransformDataMapsCountryAndContinent(t *testing.T) {
	t.Parallel()

	var input bytes.Buffer
	input.WriteString(`{"network":"1.1.1.0/24","country_code":"AU","country":"Australia","continent_code":"OC","continent":"Oceania","asn":"AS13335"}` + "\n")

	data, err := transformData(input, 1)
	if err != nil {
		t.Fatalf("transformData returned an error: %v", err)
	}

	expected := map[string]mmdbtype.Map{
		"1.1.1.0/24": {
			"continent": mmdbtype.Map{
				"code": mmdbtype.String("OC"),
				"names": mmdbtype.Map{
					"en": mmdbtype.String("Oceania"),
				},
			},
			"country": mmdbtype.Map{
				"iso_code": mmdbtype.String("AU"),
				"names": mmdbtype.Map{
					"en": mmdbtype.String("Australia"),
				},
			},
		},
	}
	if !reflect.DeepEqual(data, expected) {
		t.Errorf("data = %#v, want %#v", data, expected)
	}
}

func TestTransformDataReturnsErrorForMalformedJSON(t *testing.T) {
	t.Parallel()

	var input bytes.Buffer
	input.WriteString("not json\n")

	if _, err := transformData(input, 1); err == nil {
		t.Fatal("transformData returned no error")
	}
}

func TestNetworkCIDR(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		network string
		want    string
		wantErr string
	}{
		{
			name:    "network with prefix length",
			network: "1.1.1.0/24",
			want:    "1.1.1.0/24",
		},
		{
			name:    "ipv6 network with prefix length",
			network: "2600::/16",
			want:    "2600::/16",
		},
		{
			name:    "bare ipv4 address",
			network: "8.8.8.8",
			want:    "8.8.8.8/32",
		},
		{
			name:    "bare ipv6 address",
			network: "2001:db8::",
			wantErr: "prefix length",
		},
		{
			name:    "not an address",
			network: "not-an-ip",
			wantErr: "invalid network",
		},
		{
			name:    "empty network",
			network: "",
			wantErr: "invalid network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := networkCIDR(tt.network)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("networkCIDR(%q) returned no error", tt.network)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("networkCIDR(%q) returned an error: %v", tt.network, err)
			}
			if got != tt.want {
				t.Errorf("networkCIDR(%q) = %q, want %q", tt.network, got, tt.want)
			}
		})
	}
}

func TestWriterSkipsReservedNetworks(t *testing.T) {
	t.Parallel()

	data := map[string]mmdbtype.Map{
		"1.1.1.0/24":      countryRecord("OC", "Oceania", "AU", "Australia"),
		"2001:4:112::/48": countryRecord("NA", "North America", "US", "United States"),
	}

	output := filepath.Join(t.TempDir(), "output.mmdb")
	if err := writer(data, output); err != nil {
		t.Fatalf("writer returned an error: %v", err)
	}

	_, records := readDatabase(t, output)

	if _, ok := records["2001:4:112::/48"]; ok {
		t.Error("reserved network 2001:4:112::/48 should not be in the database")
	}
	if _, ok := records["1.1.1.0/24"]; !ok {
		t.Error("expected network 1.1.1.0/24 to be in the database")
	}
}

func TestWriterReturnsErrorForInvalidNetwork(t *testing.T) {
	t.Parallel()

	data := map[string]mmdbtype.Map{
		"not-a-network": countryRecord("NA", "North America", "US", "United States"),
	}

	output := filepath.Join(t.TempDir(), "output.mmdb")
	err := writer(data, output)
	if err == nil {
		t.Fatal("writer returned no error")
	}
	if !strings.Contains(err.Error(), "invalid network") {
		t.Errorf("error = %q, want it to contain %q", err, "invalid network")
	}
}
