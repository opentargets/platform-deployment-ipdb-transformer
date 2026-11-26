package cmd

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

// The reasons why a source record could not be published in the output
// database.
const (
	skipReasonMalformed = "malformed"
	skipReasonInvalid   = "invalid"
	skipReasonReserved  = "reserved"
	skipReasonAliased   = "aliased"
)

// skipReasonOrder is the order the skip reasons are reported in.
var skipReasonOrder = []string{
	skipReasonMalformed,
	skipReasonInvalid,
	skipReasonReserved,
	skipReasonAliased,
}

type MyData struct {
	Network       string `json:"network"`
	CountryCode   string `json:"country_code"`
	Country       string `json:"country"`
	ContinentCode string `json:"continent_code"`
	Continent     string `json:"continent"`
	ASN           string `json:"asn"`
	AsName        string `json:"as_name"`
	AsDomain      string `json:"as_domain"`
}

// skipStats keeps track of the source records that could not be published in
// the output database. The source data is not under our control, so records
// that cannot be published are skipped and counted instead of failing the
// whole transformation.
type skipStats struct {
	sourceRecords int
	skipped       int
	written       int
	noPrefix      int
	reasons       map[string]int
}

func newSkipStats() *skipStats {
	return &skipStats{reasons: map[string]int{}}
}

// skip records a source record that could not be published.
func (s *skipStats) skip(reason string) {
	s.skipped++
	s.reasons[reason]++
}

// rate is the fraction of the source records that could not be published.
func (s *skipStats) rate() float64 {
	if s.sourceRecords == 0 {
		return 0
	}
	return float64(s.skipped) / float64(s.sourceRecords)
}

// breakdown describes the skipped records by reason.
func (s *skipStats) breakdown() string {
	parts := make([]string, 0, len(s.reasons))

	for _, reason := range skipReasonOrder {
		if count := s.reasons[reason]; count > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", reason, count))
		}
	}

	others := make([]string, 0)
	for reason, count := range s.reasons {
		if count > 0 && !slices.Contains(skipReasonOrder, reason) {
			others = append(others, fmt.Sprintf("%s=%d", reason, count))
		}
	}
	sort.Strings(others)

	return strings.Join(append(parts, others...), ", ")
}

// summary describes how much of the source data was published.
func (s *skipStats) summary() string {
	skipped := "no records skipped"
	if s.skipped > 0 {
		skipped = fmt.Sprintf(
			"skipped %d of %d source records (%.4f%%): %s",
			s.skipped,
			s.sourceRecords,
			s.rate()*100,
			s.breakdown(),
		)
	}

	summary := fmt.Sprintf("%s; wrote %d networks", skipped, s.written)
	if s.noPrefix > 0 {
		summary += fmt.Sprintf("; %d bare IPv6 addresses written as /128", s.noPrefix)
	}

	return summary
}

func fromFile(input_path string, output string, maxSkipRate float64) (string, error) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("start transforming IP database")

	buffer, err := readFromFile(input_path)
	if err != nil {
		return "", fmt.Errorf("error reading the ip database from file: %w", err)
	}

	if err := transformAndWrite(buffer, output, maxSkipRate); err != nil {
		return "", err
	}

	return "transformed file", nil
}

func fromUrl(url string, output string, maxSkipRate float64) (string, error) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("start transforming IP database")

	buffer, err := readFromUrl(url)
	if err != nil {
		return "", err
	}

	if err := transformAndWrite(buffer, output, maxSkipRate); err != nil {
		return "", err
	}

	return "transformed url", nil
}

// transformAndWrite transforms the source data into a MaxMind database. The
// source data is not under our control, so individual records that cannot be
// published are skipped and counted. The transformation only fails when the
// source data cannot be used at all or when too many records had to be
// skipped.
func transformAndWrite(buffer bytes.Buffer, output string, maxSkipRate float64) error {
	stats := newSkipStats()

	ipData := transformData(buffer, stats)

	if err := checkSourceNotEmpty(stats); err != nil {
		return err
	}

	tree, err := buildTree(ipData, stats)
	if err != nil {
		return fmt.Errorf("error writing the file: %w", err)
	}

	if err := checkNetworksPublished(stats); err != nil {
		return err
	}

	if err := checkSkipRate(stats, maxSkipRate); err != nil {
		return err
	}

	if err := writeTree(tree, output); err != nil {
		return fmt.Errorf("error writing the file: %w", err)
	}

	log.Printf("%s", stats.summary())
	log.Printf("the %s database has now been created\n", output)

	return nil
}

// transformData converts the JSONL source data into the MaxMind schema.
// Records that cannot be decoded are skipped and counted instead of failing
// the transformation.
func transformData(buffer bytes.Buffer, stats *skipStats) map[string]mmdbtype.Map {
	log.Println("transforming the data to MaxMind schema")

	ipData := make(map[string]mmdbtype.Map)

	for i, line := range bytes.Split(buffer.Bytes(), []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		stats.sourceRecords++

		var data MyData

		if err := json.Unmarshal(line, &data); err != nil {
			stats.skip(skipReasonMalformed)
			log.Printf("skipping malformed record at line %d: %v\n", i+1, err)
			continue
		}

		//transform into mmdbtype
		ipData[data.Network] = mmdbtype.Map{
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

	return ipData
}

// networkCIDR returns the CIDR notation to use for a network entry. Networks
// with an explicit prefix length are used as they are, bare IPv4 addresses are
// treated as /32 networks and bare IPv6 addresses as /128 networks. The second
// return value reports whether the network was a bare IPv6 address.
func networkCIDR(cidr string) (string, bool, error) {
	if strings.Contains(cidr, "/") {
		return cidr, false, nil
	}

	ip := net.ParseIP(cidr)
	if ip == nil {
		return "", false, fmt.Errorf("invalid IP address %q", cidr)
	}
	if ip.To4() != nil {
		return cidr + "/32", false, nil
	}

	return cidr + "/128", true, nil
}

// buildTree inserts the networks into a MaxMind tree. Networks that cannot be
// published, e.g. because they belong to a reserved range, are skipped and
// counted instead of failing the transformation.
func buildTree(ipData map[string]mmdbtype.Map, stats *skipStats) (*mmdbwriter.Tree, error) {
	tree, err := mmdbwriter.New(mmdbwriter.Options{
		// "DatabaseType" is some arbitrary string describing the database.
		// At MaxMind we use strings like "GeoIP2-City", "GeoIP2-Country", etc.
		DatabaseType: "GeoIP2-Country",

		// "Description" is a map where the keys are language codes and the
		// values are descriptions of the database in that language.
		Description: map[string]string{
			"en": "Opentargets database of IP data",
		},

		// "IPVersion" can be either 4 or 6.
		IPVersion: 6,
	})

	if err != nil {
		return nil, err
	}

	log.Println("writing data into database")

	for cidr, data := range ipData {
		cidr2, bareIPv6, err := networkCIDR(cidr)
		if err != nil {
			stats.skip(skipReasonInvalid)
			log.Printf("skipping invalid network %q: %v\n", cidr, err)
			continue
		}

		_, network, err := net.ParseCIDR(cidr2)
		if err != nil {
			stats.skip(skipReasonInvalid)
			log.Printf("skipping invalid network %q: %v\n", cidr, err)
			continue
		}

		if err := tree.Insert(network, data); err != nil {
			var reservedErr *mmdbwriter.ReservedNetworkError
			if errors.As(err, &reservedErr) {
				stats.skip(skipReasonReserved)
				log.Printf("skipping reserved network %q: %v\n", cidr, err)
				continue
			}

			var aliasedErr *mmdbwriter.AliasedNetworkError
			if errors.As(err, &aliasedErr) {
				stats.skip(skipReasonAliased)
				log.Printf("skipping aliased network %q: %v\n", cidr, err)
				continue
			}

			return nil, fmt.Errorf("error inserting network %q: %w", cidr, err)
		}

		if bareIPv6 {
			stats.noPrefix++
		}
		stats.written++
	}

	return tree, nil
}

// writeTree writes the MaxMind tree to the output path.
func writeTree(tree *mmdbwriter.Tree, output_path string) error {
	fh, err := os.Create(output_path)
	if err != nil {
		return err
	}
	defer fh.Close()

	if _, err := tree.WriteTo(fh); err != nil {
		return err
	}

	return fh.Close()
}

// checkSourceNotEmpty makes sure the source data contains records. Publishing
// a database built from an empty source would be worse than failing the
// transformation.
func checkSourceNotEmpty(stats *skipStats) error {
	if stats.sourceRecords == 0 {
		return errors.New("the source data contains no records")
	}

	return nil
}

// checkNetworksPublished makes sure the source data produced a database with
// usable content. A source where every record had to be skipped is an error.
func checkNetworksPublished(stats *skipStats) error {
	if stats.written == 0 {
		return fmt.Errorf(
			"no usable networks found in the source data: all %d source records were skipped (%s)",
			stats.sourceRecords,
			stats.breakdown(),
		)
	}

	return nil
}

// checkSkipRate fails the transformation when too many of the source records
// had to be skipped. A negative maximum disables the check.
func checkSkipRate(stats *skipStats, maxSkipRate float64) error {
	if maxSkipRate < 0 {
		return nil
	}

	if stats.rate()*100 > maxSkipRate {
		return fmt.Errorf(
			"skipped %.4f%% of the source records (%d of %d), which exceeds the maximum allowed skip rate of %g%%",
			stats.rate()*100,
			stats.skipped,
			stats.sourceRecords,
			maxSkipRate,
		)
	}

	return nil
}

func readFromFile(path string) (bytes.Buffer, error) {

	log.Println("reading from file")

	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
		return bytes.Buffer{}, err
	}
	defer file.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		return bytes.Buffer{}, err
	}

	return buf, nil
}

func readFromUrl(url string) (bytes.Buffer, error) {

	log.Println("reading from URL")

	resp, err := http.Get(url)
	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("error fetching the ip database from the url %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return bytes.Buffer{}, fmt.Errorf("error fetching the ip database from the url %d", resp.StatusCode)
	}

	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("error uncompressing the file: %w", err)
	}
	defer gzipReader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, gzipReader); err != nil {
		return bytes.Buffer{}, fmt.Errorf("error creating a copy of the reader %s", err)
	}

	return buf, nil

}
