BINARY_NAME=app
INSTALL_PATH=/usr/bin/
TAG=$(shell git describe --abbrev=0 2> /dev/null || echo "0.0.1")
HASH=$(shell git rev-parse --verify --short HEAD)
VERSION="${TAG}-${HASH}"

${BINARY_NAME}:
	@printf "building version %s, stripped\n" "${VERSION}"
	@CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build \
		-ldflags "-X main.version=${VERSION} -s -w" \
		-o ${BINARY_NAME} \
		cmd/main.go

debug: ${BINARY_NAME}
	@printf "building version %s\n with debug tables" "${VERSION}"
	@CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build \
		-ldflags "-X main.version=${VERSION}" \
		-o ${BINARY_NAME} \
		cmd/main.go

install: ${BINARY_NAME}
	@sudo install ${BINARY_NAME} -v -m 0770 -o root -g root -t ${INSTALL_PATH}

clean:
	@go clean
	@rm -fv ${BINARY_NAME}