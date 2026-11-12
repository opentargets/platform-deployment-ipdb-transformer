package cmd

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/oschwald/maxminddb-golang/v2"
)

const sampleJSONL = `{"network":"1.1.1.0/24","country_code":"AU","country":"Australia","continent_code":"OC","continent":"Oceania","asn":"AS13335","as_name":"CLOUDFLARENET","as_domain":"cloudflare.com"}
{"network":"8.8.8.8","country_code":"US","country":"United States","continent_code":"NA","continent":"North America","asn":"AS15169","as_name":"GOOGLE","as_domain":"google.com"}
{"network":"2600::/16","country_code":"JP","country":"Japan","continent_code":"AS","continent":"Asia","asn":"AS7018","as_name":"NTT","as_domain":"ntt.com"}
`

type geoIP2Country struct {
	Continent struct {
		Code  string            `maxminddb:"code"`
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"continent"`
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
}

func writeInputFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "input.jsonl")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write input file: %v", err)
	}
	return path
}

func readDatabase(t *testing.T, path string) (*maxminddb.Reader, map[string]geoIP2Country) {
	t.Helper()

	reader, err := maxminddb.Open(path)
	if err != nil {
		t.Fatalf("open output database: %v", err)
	}
	t.Cleanup(func() {
		if err := reader.Close(); err != nil {
			t.Errorf("close output database: %v", err)
		}
	})

	records := make(map[string]geoIP2Country)
	for result := range reader.Networks() {
		var record geoIP2Country
		if err := result.Decode(&record); err != nil {
			t.Fatalf("decode %s: %v", result.Prefix(), err)
		}
		records[result.Prefix().String()] = record
	}
	return reader, records
}

func expectedCountry(continentCode, continent, countryCode, country string) geoIP2Country {
	var record geoIP2Country
	record.Continent.Code = continentCode
	record.Continent.Names = map[string]string{"en": continent}
	record.Country.ISOCode = countryCode
	record.Country.Names = map[string]string{"en": country}
	return record
}

func gzipContent(t *testing.T, content string) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write([]byte(content)); err != nil {
		t.Fatalf("write gzip content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}
