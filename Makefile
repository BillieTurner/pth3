dev:
	reflex -R '^logs/' -R '^testdata/' -R '^tools/' -s go run main.go
build:
	GOARCH=amd64 GOOS=linux go build -o ./build/pth3
clean:
	rm -rf ./build/*
test:
	go test -v -cover ./...

.PHONY: dev build build-linux clean test
