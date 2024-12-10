.PHONY: build run clean

build:
	templ generate ./srv/components
	go build -o dndbotwww ./srv

run: build
	./dndbotwww

clean:
	rm -f dndbotwww

fmt:
	find . -name '*.go' -exec gofumpt -w -s -extra {} \;