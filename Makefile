OUTPUT_NAME=tcpbug

build-release:
	go build -o $(OUTPUT_NAME) --ldflags '-extldflags "-static"'

build-dev:
	go build -o $(OUTPUT_NAME) 

clean:
	rm $(OUTPUT_NAME)
