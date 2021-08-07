the-hundred-combined-table: *.go
	@go build

build: the-hundred-combined-table

preview: build
	@./the-hundred-combined-table

-include *.mk
