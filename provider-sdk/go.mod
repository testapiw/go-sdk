module github.com/testapiw/go-sdk/provider-sdk

go 1.22

require github.com/testapiw/go-sdk/http-sdk v0.0.0

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/testapiw/go-sdk/http-sdk => ../http-sdk
