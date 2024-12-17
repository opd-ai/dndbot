.PHONY: build run clean

build:
	go build -o dndbotwww ./srv

run: fmt build
	killall dndbotwww; true
	./dndbotwww

clean:
	rm -frv dndbot dndbotwww profile outputs payments paywallet tmp *.log

fmt:
	find . -name '*.go' -exec gofumpt -w -s -extra {} \;

doc:
	find srv/ui/ -name '*.go' -exec code2prompt --template ~/code2prompt/templates/document-the-code.hbs --output {}.md {} \;

fox:
	rm -rf profile
	mkdir profile
	firefox --profile profile http://localhost:3000

docker:
	docker build -t dndbot .

docker-run:
	docker run -e CLAUDE_API_KEY=$(CLAUDE_API_KEY) -e HORDE_API_KEY=$(HORDE_API_KEY) --restart=always --net=host --name dndbot dndbot