app: templ
	go build main.go -o app

serve: templ
	go run .

templ:
	templ generate

.PHONY: app templ serve
