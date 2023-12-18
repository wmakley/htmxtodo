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

test: templ
	dbmate --env-file .env.test up
	go test -v htmxtodo/internal/app

.PHONY: templ serve clean test
