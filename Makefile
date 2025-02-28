.PHONY: all build run test clean

all: build

build:
ifeq ($(OS),Windows_NT)
	@echo "Билдим под Windows..."
	if not exist bin mkdir bin
	go build -o bin/factbuffer.exe ./cmd
else
	@echo "Билдим..."
	@mkdir -p bin
	go build -o bin/factbuffer ./cmd
endif

run: build
	@echo "Запускаеется..."
	./bin/factbuffer

test:
	@echo "Запускаются тесты..."
	go test -v ./...

clean:
	@echo "Чистим..."
ifeq ($(OS),Windows_NT)
	@rmdir /S /Q bin
else
	@rm -rf bin
endif
