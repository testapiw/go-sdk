module github.com/testapiw/go-sdk/provider-sdk

go 1.22

require (
	github.com/sony/gobreaker v1.0.0
	github.com/testapiw/go-sdk/http-sdk v0.0.0
)

replace github.com/testapiw/go-sdk/http-sdk => ../http-sdk
