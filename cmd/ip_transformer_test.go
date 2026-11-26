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

	stats := newSkipStats()
	data := transformData(input, stats)

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

	if stats.sourceRecords != 1 {
		t.Errorf("source records = %d, want 1", stats.sourceRecords)
	}
	if stats.skipped != 0 {
		t.Errorf("skipped = %d, want 0", stats.skipped)
	}
}

func TestTransformDataSkipsMalformedRecords(t *testing.T) {
	t.Parallel()

	var input bytes.Buffer
	input.WriteString("not json\n")
	input.WriteString("\n")
	input.WriteString(`{"network":"1.1.1.0/24"}` + "\n")

	stats := newSkipStats()
	data := transformData(input, stats)

	if len(data) != 1 {
		t.Errorf("data = %#v, want a single network", data)
	}
	if stats.sourceRecords != 2 {
		t.Errorf("source records = %d, want 2", stats.sourceRecords)
	}
	if stats.skipped != 1 {
		t.Errorf("skipped = %d, want 1", stats.skipped)
	}
	if stats.reasons[skipReasonMalformed] != 1 {
		t.Errorf("malformed = %d, want 1", stats.reasons[skipReasonMalformed])
	}
}

func TestNetworkCIDR(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		network  string
		want     string
		bareIPv6 bool
		wantErr  string
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
			name:     "bare ipv6 address",
			network:  "2a00:2000:0:8080::66",
			want:     "2a00:2000:0:8080::66/128",
			bareIPv6: true,
		},
		{
			name:    "not an address",
			network: "not-an-ip",
			wantErr: "invalid IP address",
		},
		{
			name:    "empty network",
			network: "",
			wantErr: "invalid IP address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, bareIPv6, err := networkCIDR(tt.network)
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
			if bareIPv6 != tt.bareIPv6 {
				t.Errorf("networkCIDR(%q) bare IPv6 = %t, want %t", tt.network, bareIPv6, tt.bareIPv6)
			}
		})
	}
}

func TestBuildTreeSkipsReservedNetworks(t *testing.T) {
	t.Parallel()

	data := map[string]mmdbtype.Map{
		"1.1.1.0/24":      countryRecord("OC", "Oceania", "AU", "Australia"),
		"2001:4:112::/48": countryRecord("NA", "North America", "US", "United States"),
	}

	stats := newSkipStats()
	tree, err := buildTree(data, stats)
	if err != nil {
		t.Fatalf("buildTree returned an error: %v", err)
	}
	if tree == nil {
		t.Fatal("buildTree returned no tree")
	}

	if stats.written != 1 {
		t.Errorf("written = %d, want 1", stats.written)
	}
	if stats.skipped != 1 {
		t.Errorf("skipped = %d, want 1", stats.skipped)
	}
	if stats.reasons[skipReasonReserved] != 1 {
		t.Errorf("reserved = %d, want 1", stats.reasons[skipReasonReserved])
	}
}

func TestBuildTreeSkipsInvalidNetworks(t *testing.T) {
	t.Parallel()

	data := map[string]mmdbtype.Map{
		"1.1.1.0/24":    countryRecord("OC", "Oceania", "AU", "Australia"),
		"not-a-network": countryRecord("NA", "North America", "US", "United States"),
	}

	stats := newSkipStats()
	if _, err := buildTree(data, stats); err != nil {
		t.Fatalf("buildTree returned an error: %v", err)
	}

	if stats.written != 1 {
		t.Errorf("written = %d, want 1", stats.written)
	}
	if stats.reasons[skipReasonInvalid] != 1 {
		t.Errorf("invalid = %d, want 1", stats.reasons[skipReasonInvalid])
	}
}

func TestBuildTreeCountsBareIPv6Addresses(t *testing.T) {
	t.Parallel()

	data := map[string]mmdbtype.Map{
		"2a00:2000:0:8080::66": countryRecord("EU", "Europe", "FR", "France"),
	}

	stats := newSkipStats()
	if _, err := buildTree(data, stats); err != nil {
		t.Fatalf("buildTree returned an error: %v", err)
	}

	if stats.written != 1 {
		t.Errorf("written = %d, want 1", stats.written)
	}
	if stats.noPrefix != 1 {
		t.Errorf("bare IPv6 addresses = %d, want 1", stats.noPrefix)
	}
}

func TestSkipStatsRateAndBreakdown(t *testing.T) {
	t.Parallel()

	stats := newSkipStats()
	stats.sourceRecords = 1000
	stats.skip(skipReasonMalformed)
	stats.skip(skipReasonInvalid)
	stats.skip(skipReasonReserved)

	if stats.rate() != 0.003 {
		t.Errorf("rate = %v, want 0.003", stats.rate())
	}

	breakdown := stats.breakdown()
	for _, want := range []string{"malformed=1", "invalid=1", "reserved=1"} {
		if !strings.Contains(breakdown, want) {
			t.Errorf("breakdown = %q, want it to contain %q", breakdown, want)
		}
	}

	summary := stats.summary()
	for _, want := range []string{"skipped 3 of 1000 source records", "0.3000%"} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary = %q, want it to contain %q", summary, want)
		}
	}
}

func TestSkipStatsSummaryWithoutSkippedRecords(t *testing.T) {
	t.Parallel()

	stats := newSkipStats()
	stats.sourceRecords = 10
	stats.written = 10

	summary := stats.summary()
	for _, want := range []string{"no records skipped", "wrote 10 networks"} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary = %q, want it to contain %q", summary, want)
		}
	}
}

func TestCheckSourceNotEmpty(t *testing.T) {
	t.Parallel()

	err := checkSourceNotEmpty(newSkipStats())
	if err == nil {
		t.Fatal("checkSourceNotEmpty returned no error")
	}
	if !strings.Contains(err.Error(), "no records") {
		t.Errorf("error = %q, want it to contain %q", err, "no records")
	}

	stats := newSkipStats()
	stats.sourceRecords = 1
	if err := checkSourceNotEmpty(stats); err != nil {
		t.Errorf("checkSourceNotEmpty returned an error: %v", err)
	}
}

func TestCheckNetworksPublished(t *testing.T) {
	t.Parallel()

	t.Run("every record skipped", func(t *testing.T) {
		t.Parallel()

		stats := newSkipStats()
		stats.sourceRecords = 3
		stats.skip(skipReasonReserved)

		err := checkNetworksPublished(stats)
		if err == nil {
			t.Fatal("checkNetworksPublished returned no error")
		}
		if !strings.Contains(err.Error(), "no usable networks") {
			t.Errorf("error = %q, want it to contain %q", err, "no usable networks")
		}
	})

	t.Run("usable source", func(t *testing.T) {
		t.Parallel()

		stats := newSkipStats()
		stats.sourceRecords = 3
		stats.written = 2

		if err := checkNetworksPublished(stats); err != nil {
			t.Errorf("checkNetworksPublished returned an error: %v", err)
		}
	})
}

func TestCheckSkipRate(t *testing.T) {
	t.Parallel()

	stats := newSkipStats()
	stats.sourceRecords = 1000
	stats.skip(skipReasonReserved)

	if err := checkSkipRate(stats, 1); err != nil {
		t.Errorf("checkSkipRate returned an error: %v", err)
	}

	err := checkSkipRate(stats, 0)
	if err == nil {
		t.Fatal("checkSkipRate returned no error")
	}
	if !strings.Contains(err.Error(), "exceeds the maximum allowed skip rate") {
		t.Errorf("error = %q, want it to contain %q", err, "exceeds the maximum allowed skip rate")
	}

	if err := checkSkipRate(stats, -1); err != nil {
		t.Errorf("checkSkipRate returned an error: %v", err)
	}
}

func TestWriteTreeCreatesDatabase(t *testing.T) {
	t.Parallel()

	stats := newSkipStats()
	tree, err := buildTree(map[string]mmdbtype.Map{
		"1.1.1.0/24": countryRecord("OC", "Oceania", "AU", "Australia"),
	}, stats)
	if err != nil {
		t.Fatalf("buildTree returned an error: %v", err)
	}

	output := filepath.Join(t.TempDir(), "output.mmdb")
	if err := writeTree(tree, output); err != nil {
		t.Fatalf("writeTree returned an error: %v", err)
	}

	_, records := readDatabase(t, output)
	if _, ok := records["1.1.1.0/24"]; !ok {
		t.Error("expected network 1.1.1.0/24 to be in the database")
	}
}
