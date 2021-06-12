prog-name = ./dist/tmai.server.mock

build:
	go build -o $(prog-name) .

install:
	go install -o $(prog-name) .

clean:
	rm $(prog-name)

run: build
	$(prog-name) tmai
