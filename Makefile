GOBUILD=go build

.PHONY: all clean

all:
	mkdir -p bin
	$(GOBUILD) -o bin/apiserver .
	$(GOBUILD) -o bin/worker ./worker
	$(GOBUILD) -o bin/tester ./sendOrder

clean:
	rm -fr ./bin
