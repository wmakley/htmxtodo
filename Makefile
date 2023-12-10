build/app: templ
	mkdir -p build
	go build -o $@ .

serve: templ
	go run .

templ:
	templ generate

clean:
	rm build/app

.PHONY: templ serve clean
