GOBUILD=go build

.PHONY: all clean

all:
	$(GOBUILD) -o bin/apiserver.exe .
	$(GOBUILD) -o bin/worker.exe ./worker
	$(GOBUILD) -o bin/tester.exe ./sendOrder

clean:
	rm ./bin
