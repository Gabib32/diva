# Makefile used to create packages for diva.
.NOTPARALLEL:

GO_PACKAGE_PREFIX := github.com/clearlinux/diva

.PHONY: build install clean check

.DEFAULT_GOAL := build


build:
	go install ${GO_PACKAGE_PREFIX}

install:
	test -d $(DESTDIR)/usr/bin || install -D -d -m 00755 $(DESTDIR)/usr/bin;
	install -m 00755 $(GOPATH)/bin/diva $(DESTDIR)/usr/bin/.

check:
	go test -cover ${GO_PACKAGE_PREFIX}/...

# TODO: since Go 1.10 we have support for passing multiple packages
# to coverprofile. Update this to work on all packages.
.PHONY: checkcoverage
checkcoverage:
	go test -cover ${GO_PACKAGE_PREFIX}/... -coverprofile=coverage.out
	go tool cover -html=coverage.out

.PHONY: lint
lint:
	@gometalinter.v2 --deadline=10m --tests --vendor --disable-all \
	--enable=misspell \
	--enable=vet \
	--enable=ineffassign \
	--enable=gofmt \
	--enable=gocyclo --cyclo-over=15 \
	--enable=golint \
	--enable=deadcode \
	--enable=varcheck \
	--enable=structcheck \
	--enable=unused \
	--enable=vetshadow \
	--enable=errcheck \
	./...

clean:
	go clean -i -x
	rm -f diva-*.tar.gz

release:
	@if [ ! -d .git ]; then \
		echo "Release needs to be used from a git repository"; \
		exit 1; \
	fi
	@VERSION=$$(grep -e 'const version' cmd/root.go | cut -d '"' -f 2) ; \
	if [ -z "$$VERSION" ]; then \
		echo "Couldn't extract version number from the source code"; \
		exit 1; \
	fi; \
	git archive --format=tar.gz --verbose -o diva-$$VERSION.tar.gz HEAD --prefix=diva-$$VERSION/
