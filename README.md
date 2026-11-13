# Open Targets Platform IP Database Transformation

[![ci](https://github.com/opentargets/platform-deployment-ipdb-transformer/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/opentargets/platform-deployment-ipdb-transformer/actions/workflows/ci.yaml?query=branch%3Amain)

The IP Database Transformation tool transforms from [IPInfo](https://ipinfo.io/) schema to [MaxMind](https://www.maxmind.com/en/geoip-databases) schema.

Given That frafana alloy ip resolution works with MaxMind schema we get a database in IPInfo schema and transform it to MaxMind schema.

## Requirements.

- [go](https://go.dev/) >= 1.25.0

## Input format

The input must be a JSON Lines (JSONL) file: each line must contain one standalone JSON object. The supported fields are:

| Field | Required | Description |
| --- | --- | --- |
| `network` | Yes | IPv4 or IPv6 network in CIDR notation, such as `1.1.1.0/24` or `2001:db8::/32`. |
| `country_code` | No | ISO 3166-1 alpha-2 country code, such as `US`. |
| `country` | No | Country name in English, such as `United States`. |
| `continent_code` | No | Continent code, such as `NA`. |
| `continent` | No | Continent name in English, such as `North America`. |
| `asn` | No | Autonomous system number. Accepted but not written to the output database. |
| `as_name` | No | Autonomous system name. Accepted but not written to the output database. |
| `as_domain` | No | Autonomous system domain. Accepted but not written to the output database. |

Example:

```jsonl
{"network":"1.1.1.0/24","country_code":"AU","country":"Australia","continent_code":"OC","continent":"Oceania","asn":"AS13335","as_name":"CLOUDFLARENET","as_domain":"cloudflare.com"}
{"network":"8.8.8.8","country_code":"US","country":"United States","continent_code":"NA","continent":"North America","asn":"AS15169","as_name":"GOOGLE","as_domain":"google.com"}
```

IPv4 networks must include an explicit prefix length; a bare IPv4 address is treated as a `/32` network. IPv6 networks must include an explicit prefix length.

## Output format

The output is a MaxMind database (`.mmdb`) with the database type `GeoIP2-Country`. It supports both IPv4 and IPv6 networks. Each network contains the following data:

```json
{
  "continent": {
    "code": "OC",
    "names": {
      "en": "Oceania"
    }
  },
  "country": {
    "iso_code": "AU",
    "names": {
      "en": "Australia"
    }
  }
}
```

| Output path | Input field |
| --- | --- |
| `continent.code` | `continent_code` |
| `continent.names.en` | `continent` |
| `country.iso_code` | `country_code` |
| `country.names.en` | `country` |

The ASN fields (`asn`, `as_name`, and `as_domain`) are not included in the output database.

## Usage

To build the application you just need to run `make build` or just `make`. And to clear to artifacts you can use `make clean`.

Once the application is built you can find help for the commands using:

``` bash
./ipdb_transformer help
```

This is the list of commnads:

``` bash
  completion  Generate the autocompletion script for the specified shell
  file        Transform json file from local path to mmdb
  help        Help about any command
  url         Transform json file from url to mmdb
```

### Transform from file

If you have a local file containing an IPInfo database in JSONL format, you can use the `file` command. This command accepts two positional arguments: `database_path` and `output`. The `database_path` is the path to the uncompressed IPInfo JSONL file, and `output` is the optional output path for the MaxMind database; it defaults to `output.mmdb`. This results in the following command:

``` bash
./ipdb_transformer file <database_path> <output>
```

### Transform from URL

You can use the `url` command to transform an IPInfo database from a valid URL. The URL must return HTTP 200 with a gzip-compressed JSONL response using the input format described above. The tool automatically decompresses the response before processing it.

This command accepts two positional arguments: `database_url` and `output`. The `database_url` is the URL of the gzip-compressed IPInfo JSONL file, and `output` is the optional local output path for the MaxMind database; it defaults to `output.mmdb`. This results in the following command:

``` bash
./ipdb_transformer url <database_url> <output>
```

You can generate shell completions by running one of these three options:

```
$ source <(./ipdb_transformer completion {bash,fish,zsh})
```

## Copyright

Copyright 2014-2025 EMBL - European Bioinformatics Institute, Genentech, GSK,
MSD, Pfizer, Sanofi and Wellcome Sanger Institute

This software was developed as part of the Open Targets project. For more
information please see: http://www.opentargets.org

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this file except in compliance with the License. You may obtain a copy of the
License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
