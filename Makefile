build/app: templ
	mkdir -p build
	go build -o $@ .

tmp/main: templ
	go build -o $@ .

serve: templ
	go run .

templ:
	templ generate

clean:
	rm -fv build/app tmp/main

.PHONY: templ serve clean
