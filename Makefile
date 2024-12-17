.PHONY: build run clean

build:
	go build -o dndbotwww ./srv

run: build
	killall dndbotwww; true
	./dndbotwww

clean:
	rm -frv dndbotwww srv/components/*.go

fmt:
	find . -name '*.go' -exec gofumpt -w -s -extra {} \;

doc:
	find srv/ui/ -name '*.go' -exec code2prompt --template ~/code2prompt/templates/document-the-code.hbs --output {}.md {} \;

fox:
	rm -rf profile
	mkdir profile
	firefox --profile profile http://localhost:3000
