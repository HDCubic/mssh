BUILD=`date`
COMMITSHA1=`git rev-parse HEAD`

LDFLAGS=-ldflags "-X main.VERSION=${VERSION} -X 'main.BUILD=${BUILD}' -X main.COMMITSHA1=${COMMITSHA1}"

RELEASE_DIR=mssh-${VERSION}

# Builds the project
all: build

build:
	@echo $(LDFLAGS)
	mkdir -p $(RELEASE_DIR)
	go build $(LDFLAGS) -o $(RELEASE_DIR)/mssh
	cp -rf Makefile.in $(RELEASE_DIR)/Makefile
	cp -rf README.md $(RELEASE_DIR)/README.md
	tar -czvf $(RELEASE_DIR).tar.gz $(RELEASE_DIR)
	rm -rf $(RELEASE_DIR)

install: clean
	go install $(LDFLAGS)

clean:
	rm -rf $(GOPATH)/bin/mssh
	rm -rf $(RELEASE_DIR)*
