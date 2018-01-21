all: install

install:
	go install

clean:
	rm -rf $(GOPATH)/bin/mssh
