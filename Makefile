BINARY_NAME=borgbecue
INSTALL_PATH=/usr/bin/
SYSTEMD_PATH=/etc/systemd/system/
CONFIG_PATH=/etc/${BINARY_NAME}/
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
	@printf "installing compiled binary\n"
	@sudo install ${BINARY_NAME} -v -m 0770 -o root -g root -t ${INSTALL_PATH}

service: install
	@printf "\ninstalling and enabling systemd units\n"
	@echo; sudo install ${BINARY_NAME}.service -v -m 0660 -o root -g root -t ${SYSTEMD_PATH}
	@echo; sudo install ${BINARY_NAME}.timer -v -m 0660 -o root -g root -t ${SYSTEMD_PATH}
	@echo; sudo systemctl enable ${BINARY_NAME}.timer
	@echo; sudo systemctl start ${BINARY_NAME}.timer

clean:
	@go clean
	@rm -fv ${BINARY_NAME}