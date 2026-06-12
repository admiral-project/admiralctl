module github.com/admiral-project/admiral/admiralctl

go 1.21

toolchain go1.22.3

replace github.com/admiral-project/admiral/admirald => ../admirald

require (
	github.com/admiral-project/admiral/admirald v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v2 v2.4.0
)
