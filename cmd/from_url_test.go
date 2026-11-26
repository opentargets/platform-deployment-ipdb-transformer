package cmd

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFromURLProducesExpectedSchema(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(gzipContent(t, sampleJSONL))
	}))
	defer server.Close()

	output := filepath.Join(t.TempDir(), "output.mmdb")
	if _, err := fromUrl(server.URL, output, skipRateDisabled); err != nil {
		t.Fatalf("fromUrl returned an error: %v", err)
	}

	reader, records := readDatabase(t, output)
	if reader.Metadata.DatabaseType != "GeoIP2-Country" {
		t.Errorf("database type = %q, want %q", reader.Metadata.DatabaseType, "GeoIP2-Country")
	}
	if reader.Metadata.IPVersion != 6 {
		t.Errorf("IP version = %d, want 6", reader.Metadata.IPVersion)
	}

	expected := map[string]geoIP2Country{
		"1.1.1.0/24": expectedCountry("OC", "Oceania", "AU", "Australia"),
		"8.8.8.8/32": expectedCountry("NA", "North America", "US", "United States"),
		"2600::/16":  expectedCountry("AS", "Asia", "JP", "Japan"),
	}
	if !reflect.DeepEqual(records, expected) {
		t.Errorf("records = %#v, want %#v", records, expected)
	}
}

func TestFromURLReturnsErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErrMsg string
	}{
		{
			name: "non-200 status",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not found", http.StatusNotFound)
			},
			wantErrMsg: "404",
		},
		{
			name: "uncompressed response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(sampleJSONL))
			},
			wantErrMsg: "invalid header",
		},
		{
			name: "malformed JSON",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(gzipContent(t, "not json\n"))
			},
			wantErrMsg: "no usable networks",
		},
		{
			name: "corrupt gzip",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte{0x1f, 0x8b, 0x08, 0x00, 0x00})
			},
			wantErrMsg: "unexpected EOF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(tt.handler)
			defer server.Close()

			output := filepath.Join(t.TempDir(), "output.mmdb")
			_, err := fromUrl(server.URL, output, skipRateDisabled)
			if err == nil {
				t.Fatal("fromUrl returned no error")
			}
			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErrMsg)
			}
		})
	}
}

func TestFromURLReturnsErrorForEmptyGzipResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(gzipContent(t, ""))
	}))
	defer server.Close()

	output := filepath.Join(t.TempDir(), "output.mmdb")
	_, err := fromUrl(server.URL, output, skipRateDisabled)
	if err == nil {
		t.Fatal("fromUrl returned no error")
	}
	if !strings.Contains(err.Error(), "no records") {
		t.Errorf("error = %q, want it to contain %q", err, "no records")
	}
}

func TestFromURLSkipsReservedNetworks(t *testing.T) {
	t.Parallel()

	content := `{"network":"2001:4:112::/48","country_code":"US","country":"United States","continent_code":"NA","continent":"North America"}` + "\n" +
		`{"network":"1.1.1.0/24","country_code":"AU","country":"Australia","continent_code":"OC","continent":"Oceania"}` + "\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(gzipContent(t, content))
	}))
	defer server.Close()

	output := filepath.Join(t.TempDir(), "output.mmdb")
	if _, err := fromUrl(server.URL, output, skipRateDisabled); err != nil {
		t.Fatalf("fromUrl returned an error: %v", err)
	}

	_, records := readDatabase(t, output)
	if _, ok := records["2001:4:112::/48"]; ok {
		t.Error("reserved network 2001:4:112::/48 should not be in the database")
	}
	expected := map[string]geoIP2Country{
		"1.1.1.0/24": expectedCountry("OC", "Oceania", "AU", "Australia"),
	}
	if !reflect.DeepEqual(records, expected) {
		t.Errorf("records = %#v, want %#v", records, expected)
	}
}

func TestFromURLPreservesQueryString(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("token"); got != "secret" {
			t.Errorf("query token = %q, want %q", got, "secret")
		}
		_, _ = w.Write(gzipContent(t, sampleJSONL))
	}))
	defer server.Close()

	output := filepath.Join(t.TempDir(), "output.mmdb")
	if _, err := fromUrl(server.URL+"?token=secret", output, skipRateDisabled); err != nil {
		t.Fatalf("fromUrl returned an error: %v", err)
	}
}

func TestFromURLReturnsErrorForUnreachableServer(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	output := filepath.Join(t.TempDir(), "output.mmdb")
	if _, err := fromUrl(url, output, skipRateDisabled); err == nil {
		t.Fatal("fromUrl returned no error")
	}
}
