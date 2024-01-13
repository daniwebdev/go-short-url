run:
	export  GO_SHORT_KEY=abc && go run *.go --port=8020
build:
	go build *.go