module github.com/admiral-project/admiral/admiralctl

go 1.21

toolchain go1.22.3

require (
	github.com/admiral-project/admiral/admirald v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.10.2
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
)

replace github.com/admiral-project/admiral/admirald => ../admirald
