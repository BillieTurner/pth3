dev:
	reflex -R '^logs/' -R '^testdata/' -R '^tools/' -s go run main.go
build:
	GOARCH=amd64 GOOS=linux go build -o ./build/pth3-s-linux ./src/server/server.go
	GOOS=darwin GOARCH=arm64 go build -o ./build/pth3-c-mac-m1 ./src/client/client.go
clean:
	rm -rf ./build/*
test:
	go test -v -cover ./...

.PHONY: dev build build-linux clean test
