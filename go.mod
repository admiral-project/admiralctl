module github.com/admiral-project/admiral/admiralctl

go 1.26.5

require (
	github.com/admiral-project/admiral/admirald v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.10.2
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace github.com/admiral-project/admiral/admirald => ./admirald
