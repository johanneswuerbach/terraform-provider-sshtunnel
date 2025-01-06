# SSHTunnel Terraform Provider

A Terraform Provider that allows the creation of ephemeral SSH tunnels using an [ephemeral resource](https://developer.hashicorp.com/terraform/language/resources/ephemeral), which enables
connecting to database servers (e.g. AWS RDS) running inside a private network (e.g. AWS VPC) using an SSH jump host.

## Features

* Automatic forward port assignments
* Configurable retries

## Next steps

* [Usage](https://registry.terraform.io/providers/johanneswuerbach/sshtunnel/latest)
* [Documentation](https://registry.terraform.io/providers/johanneswuerbach/sshtunnel/latest/docs)

## Requirements

* [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.10

## Development

### Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

### Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

### Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
