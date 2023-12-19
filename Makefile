build/main: templ
	go build -o $@ .

serve: templ
	go run .

templ:
	templ generate

clean:
	rm -fv build/main

test: templ
	dbmate --env-file .env.test up
	go test htmxtodo/internal/app

.PHONY: templ serve clean test
