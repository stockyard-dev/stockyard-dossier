build:
	CGO_ENABLED=0 go build -o dossier ./cmd/dossier/

run: build
	./dossier

test:
	go test ./...

clean:
	rm -f dossier

.PHONY: build run test clean
