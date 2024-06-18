all: generate
	go build -o goole bin/*.go


windows:
	GOOS=windows GOARCH=amd64 \
            go build -ldflags="-s -w" \
	    -o goole.exe ./bin/*.go

generate:
	cd parser/ && binparsegen conversion.spec.yaml > ole_gen.go


test:
	go test -v ./...
