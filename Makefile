OUTPUT_NAME=tcpbug

build:
	go build -o $(OUTPUT_NAME) 

run: build
	./$(OUTPUT_NAME)

clean:
	rm $(OUTPUT_NAME)
