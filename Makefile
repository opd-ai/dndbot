.PHONY: build run clean

build:
	mkdir -p web
	GOARCH=wasm GOOS=js go build -o web/app.wasm ./srv
	go build -o dndbotwww ./srv

run: build
	./dndbotwww

clean:
	rm -f web/app.wasm
	rm -f dndbotwww