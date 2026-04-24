ifeq ($(OS),Windows_NT)
    SHELL=CMD.EXE
    SET=set
    WHICH=where.exe
    DEL=del
    NUL=nul
    CP=cmd /c copy
else
    SET=export
    WHICH=which
    DEL=rm
    NUL=/dev/null
    CP=cp
endif

ifndef GO
    SUPPORTGO=go1.20.14
    GO:=$(shell $(WHICH) $(SUPPORTGO) 2>$(NUL) || echo go)
endif

NAME:=$(notdir $(CURDIR))
VERSION:=$(shell git describe --tags 2>$(NUL) || echo v0.0.0)
GOOPT:=-ldflags "-s -w -X github.com/hymkor/jegan.version=$(VERSION)"
EXE:=$(shell $(GO) env GOEXE)

build:
	$(GO) fmt ./...
	$(SET) "CGO_ENABLED=0" && $(GO) build $(GOOPT) -tags debug "./cmd/jegan"

all:
	$(GO) fmt ./...
	$(SET) "CGO_ENABLED=0" && $(GO) build $(GOOPT) "./cmd/testjson"
	$(SET) "CGO_ENABLED=0" && $(GO) build $(GOOPT) "./cmd/nemo"
	$(SET) "CGO_ENABLED=0" && $(GO) build $(GOOPT) "./cmd/jegan"

test:
	$(GO) fmt ./...
	$(GO) test -v ./...

_dist:
	$(SET) "CGO_ENABLED=0" && $(GO) build $(GOOPT) "./cmd/jegan"
	zip -9 $(NAME)-$(VERSION)-$(GOOS)-$(GOARCH).zip $(NAME)$(EXE)

dist:
	$(SET) "GOOS=windows" && $(SET) "GOARCH=386"   && $(MAKE) _dist
	$(SET) "GOOS=windows" && $(SET) "GOARCH=amd64" && $(MAKE) _dist
	$(SET) "GOOS=darwin"  && $(SET) "GOARCH=amd64" && $(MAKE) _dist
	$(SET) "GOOS=darwin"  && $(SET) "GOARCH=arm64" && $(MAKE) _dist
	$(SET) "GOOS=freebsd" && $(SET) "GOARCH=amd64" && $(MAKE) _dist
	$(SET) "GOOS=linux"   && $(SET) "GOARCH=386"   && $(MAKE) _dist
	$(SET) "GOOS=linux"   && $(SET) "GOARCH=amd64" && $(MAKE) _dist

bump:
	$(GO) run github.com/hymkor/latest-notes@latest -suffix "-goinstall" -gosrc jegan CHANGELOG*.md > version.go

clean:
	$(DEL) *.zip $(NAME)$(EXE)

release:
	$(GO) run github.com/hymkor/latest-notes@latest | gh release create -d --notes-file - -t $(VERSION) $(VERSION) $(wildcard $(NAME)-$(VERSION)-*.zip)

manifest:
	$(GO) run github.com/hymkor/make-scoop-manifest@latest -all *-windows-*.zip > $(NAME).json

docs:
	$(GO) run github.com/hymkor/minipage@latest -title "Jegan - A terminal JSON editor" -outline-in-sidebar -readme-to-index README.md > docs/index.html
	$(GO) run github.com/hymkor/minipage@latest -title "Jegan - A terminal JSON editor" -outline-in-sidebar -readme-to-index README_ja.md > docs/index_ja.html
	$(GO) run github.com/hymkor/minipage@latest -title "Jegan - Changelog " -outline-in-sidebar -readme-to-index CHANGELOG.md > docs/CHANGELOG.html
	$(GO) run github.com/hymkor/minipage@latest -title "Jegan - Changelog " -outline-in-sidebar -readme-to-index CHANGELOG_ja.md > docs/CHANGELOG_ja.html
	$(CP) demo.gif "docs/demo.gif"

readme:
	$(GO) run github.com/hymkor/example-into-readme@latest
	$(GO) run github.com/hymkor/example-into-readme@latest -target README_ja.md
draft:
	$(GO) run github.com/hymkor/example-into-readme@latest -target draft/README.md
	$(GO) run github.com/hymkor/example-into-readme@latest -target draft/README_ja.md


.PHONY: all test dist _dist clean release manifest docs draft
