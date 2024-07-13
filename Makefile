build: clean
	@go build -o bin/fs

run: build
	@./bin/fs

test:
	@go test -v -race -count=1 ./...

clean:
	@rm -rf *_network/
