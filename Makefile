BINARY_NAME=borgbecue
INSTALL_PATH=/usr/bin/
SYSTEMD_PATH=/etc/systemd/system/
CONFIG_PATH=/etc/${BINARY_NAME}/
TAG=$(shell git describe --abbrev=0 2> /dev/null || echo "0.0.1")
HASH=$(shell git rev-parse --verify --short HEAD)
VERSION="${TAG}-${HASH}"
TARBALL=${BINARY_NAME}-${TAG}.tar.gz

${BINARY_NAME}:
	@printf "... building version %s, stripped\n" "${VERSION}"
	@CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build \
		-ldflags "-X main.version=${VERSION} -s -w" \
		-o ${BINARY_NAME} \
		cmd/main.go
	@echo

debug: ${BINARY_NAME}
	@printf "... building version %s with debug tables\n" "${VERSION}"
	@CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build \
		-ldflags "-X main.version=${VERSION}" \
		-o ${BINARY_NAME} \
		cmd/main.go
	@echo

release: ${BINARY_NAME}
	@printf "... building a tarball for release %s\n" "${VERSION}"
	@tar -czvf ${TARBALL} ${BINARY_NAME} ${BINARY_NAME}.service ${BINARY_NAME}.timer configs/${BINARY_NAME}.yaml
	@du -h ${TARBALL}

install: ${BINARY_NAME}
	@echo "... installing compiled binary:"
	@sudo install ${BINARY_NAME} -v -m 0770 -o root -g root -t ${INSTALL_PATH}

	@echo "... installing example configuration file:"
	@sudo mkdir -m 0770 -pv ${CONFIG_PATH};
	@sudo install configs/${BINARY_NAME}.yaml -v -m 0660 -o root -g root -t ${CONFIG_PATH}
	@echo

service: install
	@echo "... installing and enabling systemd units:"
	@sudo install ${BINARY_NAME}.service -v -m 0660 -o root -g root -t ${SYSTEMD_PATH}
	@sudo install ${BINARY_NAME}.timer -v -m 0660 -o root -g root -t ${SYSTEMD_PATH}
	@sudo systemctl enable ${BINARY_NAME}.timer
	@sudo systemctl start ${BINARY_NAME}.timer
	@echo

clean:
	@go clean
	@rm -fv ${BINARY_NAME}
	@rm -fv ${TARBALL}
	@echo