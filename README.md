# Open Targets Platform IP Database Transformation

The IP Database Transformation tool transforms from [IPInfo](https://ipinfo.io/) schema to [MaxMind](https://www.maxmind.com/en/geoip-databases) schema.

Given That frafana alloy ip resolution works with MaxMind schema we get a database in IPInfo schema and transform it to MaxMind schema.

## Requirements.

- [go](https://go.dev/) >= 1.25.0

## Usage

To build the application you just need to run: 
```
make
```

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

If you have a file with a IPInfo database you can use the `file` command. This command has two position arguments `database_path output`. The `database` is the path to the IPInfo database file and the `output` is the optional output path for the MaxMind database it's default value es `output.mmdb`. This results in the following command:

``` bash
./ipdb_transformer file <database_path> <output>
```

### Transform from url

You can use the `url` command to transforma an IPInfo database from a valid url. This command has two position arguments `database_url output`. The `database` is the path to the IPInfo database file and the `output` is the optional local path for the MaxMind database it's default value es `output.mmdb`. This results in the following command:

``` bash
./ipdb_transformer url <database_url> <output>
```

You can generate shell completions by running one of these three options:

```
$ source <(./platform completion {bash,fish,zsh})
```
