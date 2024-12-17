.PHONY: build run clean

build:
	templ generate ./srv/components
	go build -o dndbotwww ./srv

run: build
	killall dndbotwww
	./dndbotwww

clean:
	rm -frv dndbotwww srv/components/*.go

fmt:
	find . -name '*.go' -exec gofumpt -w -s -extra {} \;
	find . -name '*.templ' -exec templ fmt -w 16 {} \;

doc:
	find srv/ui/ -name '*.go' -exec code2prompt --template ~/code2prompt/templates/document-the-code.hbs --output {}.md {} \;

fox:
	rm -rf profile
	mkdir profile
	firefox --profile profile http://localhost:3000
