all: mac linux win32 win64
	rm -rf mssh mssh.exe
	mv release mssh
	tar -czvf mssh.tar.gz mssh
	rm -rf mssh

install:
	go install
mac: clean
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build
	mkdir -p release/mac
	cp -rf mssh release/mac
win64: clean
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build
	mkdir -p release/win64
	cp -rf mssh.exe release/win64
win32: clean
	CGO_ENABLED=0 GOOS=windows GOARCH=386 go build
	mkdir -p release/win32
	cp -rf mssh.exe release/win32
linux: clean
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build
	mkdir -p release/linux
	cp -rf mssh release/linux
clean:
	rm -rf mssh mssh.exe mssh.tar.gz

