.PHONY: test
test:
	@go build
	@./cmtstringer -type StatusCode ./http
	@go test ./http
