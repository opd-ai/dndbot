.PHONY: build run clean

build:
	templ generate ./srv/components
	go build -o dndbotwww ./srv

run: build
	./dndbotwww

clean:
	rm -frv dndbotwww srv/components/*.go

fmt:
	find . -name '*.go' -exec gofumpt -w -s -extra {} \;
	find . -name '*.templ' -exec templ fmt -w 16 {} \;