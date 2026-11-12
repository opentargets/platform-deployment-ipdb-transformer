package cmd

import (
	"bytes"
	"reflect"
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
