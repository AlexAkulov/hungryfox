NAME := hungryfox
GIT_TAG := $(shell git describe --always --tags --abbrev=0 | tail -c +2)
GIT_COMMIT := $(shell git rev-list v${GIT_TAG}..HEAD --count)
GO_VERSION := $(shell go version | cut -d' ' -f3)
VERSION := ${GIT_TAG}.${GIT_COMMIT}
RELEASE := 1
GO_VERSION := $(shell go version | cut -d' ' -f3)
BUILD_DATE := $(shell date --iso-8601=second)
LDFLAGS := -ldflags "-X main.version=${VERSION}-${RELEASE} -X main.goVersion=${GO_VERSION} -X main.buildDate=${BUILD_DATE}"

.PHONY: default clean prepare test test_codecov build rpm travis

default: clean test build

clean:
	rm -rf build

test:
	go test ./...

prepare:
	go get "github.com/smartystreets/goconvey"

test_codecov:
	go test -race -coverprofile="coverage.txt" ./...

build:
	mkdir -p build/root/usr/bin
	go build ${LDFLAGS} -o build/root/usr/bin/${NAME} ./cmd/hungryfox

tar:
	mkdir -p build/root/etc/${NAME}
	build/root/usr/bin/${NAME} -default-config > build/root/etc/${NAME}/config.yml
	tar -czvPf build/${NAME}-${VERSION}-${RELEASE}.tar.gz -C build/root .

rpm:
	fpm -t rpm \
		-s "tar" \
		--description "HungryFox" \
		--vendor "Alexander Akulov" \
		--url "https://github.com/AlexAkulov/hungryfox" \
		--license "MIT" \
		--name "${NAME}" \
		--version "${VERSION}" \
		--iteration "${RELEASE}" \
		--config-files "/etc/${NAME}/config.yml" \
		-p build \
		build/${NAME}-${VERSION}-${RELEASE}.tar.gz

deb:
	fpm -t deb \
		-s "tar" \
		--description "HungryFox" \
		--vendor "Alexander Akulov" \
		--url "https://github.com/AlexAkulov/hungryfox" \
		--license "MIT" \
		--name "${NAME}" \
		--version "${VERSION}" \
		--iteration "${RELEASE}" \
		--config-files "/etc/${NAME}/config.yml" \
		-p build \
		build/${NAME}-${VERSION}-${RELEASE}.tar.gz

travis: prepare test_codecov build tar rpm deb
