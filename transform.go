package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/schollz/progressbar/v3"
)

type MyData struct {
	Network       string `json:network`
	CountryCode   string `json:"country_code"`
	Country       string `json:country`
	ContinentCode string `json:"continent_code"`
	Continent     string `json:continent`
	ASN           string `json:asn`
	AsName        string `json:as_name`
	AsDomain      string `json:as_domain`
}

func readAndTransformData(path string) (map[string]mmdbtype.Map, error) {
	line_qty, err := countLinesFast(path)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("Counted", line_qty)

	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	println("Reading the data")

	bar := progressbar.Default(int64(line_qty))

	ip_data := make(map[string]mmdbtype.Map)

	for {
		var data MyData

		err := decoder.Decode(&data)

		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println(err)
			return nil, err
		}
		// fmt.Printf("the data n %s, c %s, n %s, c %s \n", data.Country, data.CountryCode, data.Continent, data.ContinentCode)
		bar.Add(1)

		//transform into mmdbtype
		ip_data[data.Network] = mmdbtype.Map{
			"continent": mmdbtype.Map{
				"code": mmdbtype.String(data.ContinentCode),
				"names": mmdbtype.Map{
					"en": mmdbtype.String(data.Continent),
				},
			},
			"country": mmdbtype.Map{
				"iso_code": mmdbtype.String(data.CountryCode),
				"names": mmdbtype.Map{
					"en": mmdbtype.String(data.Country),
				},
			},
		}
	}
	return ip_data, nil
}

func main() {
	fmt.Println("Transforming the database")

	ip_data, err := readAndTransformData("ipinfo_lite.json")

	if err != nil {
		println("Error leyendo y transformando el archivo")
	}

	writer(ip_data)

}

func countLinesFast(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	buf := make([]byte, 1<<20) // 1MB chunks
	count := 0

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			count += bytes.Count(buf[:n], []byte{'\n'})
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	// Handle files that don't end with a newline
	// (last line still counts as a record in JSONL)
	if count > 0 {
		// Quick check: if file size > 0 and last byte isn't \n, add 1
		// For simplicity, you can skip this if you control the data generation
	}

	return count, nil
}

func writer(ip_data map[string]mmdbtype.Map) {
	writer, err := mmdbwriter.New(mmdbwriter.Options{
		// "DatabaseType" is some arbitrary string describing the database.
		// At MaxMind we use strings like "GeoIP2-City", "GeoIP2-Country", etc.
		DatabaseType: "GeoIP2-Country",

		// "Description" is a map where the keys are language codes and the
		// values are descriptions of the database in that language.
		Description: map[string]string{
			"en": "My database of IP data",
		},

		// "IPVersion" can be either 4 or 6.
		IPVersion: 6,
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Inserting data into database")

	bar := progressbar.Default(int64(len(ip_data)))

	for cidr, data := range ip_data {
		var cidr2 = ""
		if strings.Contains(cidr, "/") {
			cidr2 = cidr
		} else {
			cidr2 = cidr + "/32"
		}
		_, network, err := net.ParseCIDR(cidr2)
		if err != nil {
			log.Fatal(err)
		}
		if err := writer.Insert(network, data); err != nil {
			log.Fatal(err)
		}
		bar.Add(1)
	}

	fh, err := os.Create("ip_conf.mmdb")
	if err != nil {
		log.Fatal(err)
	}
	defer fh.Close()

	_, err = writer.WriteTo(fh)
	if err != nil {
		log.Fatal(err)
	}

	if err := fh.Close(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("users.mmdb has now been created")
}
