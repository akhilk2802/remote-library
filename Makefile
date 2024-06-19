# folder name of the package of interest
PKGNAME = remote

.PHONY: build final checkpoint all final-race checkpoint-race all-race clean docs
.SILENT: build final checkpoint all final-race checkpoint-race all-race clean docs

# compile the remote library.
build:
	cd src/$(PKGNAME); go build $(PKGNAME).go

# run conformance tests.
final: build
	cd src/$(PKGNAME); go test -v -run Final

checkpoint: build
	cd src/$(PKGNAME); go test -v -run Checkpoint

all: build
	cd src/$(PKGNAME); go test -v

final-race: build
	cd src/$(PKGNAME); go test --timeout 60s -v -race -run Final

checkpoint-race: build
	cd src/$(PKGNAME); go test --timeout 60s -v -race -run Checkpoint

all-race: build
	cd src/$(PKGNAME); go test --timeout 60s -v -race 
    
# delete executable and docs, leaving only source
clean:
	rm -rf src/$(PKGNAME)/$(PKGNAME) src/$(PKGNAME)/$(PKGNAME)-doc.txt

# generate documentation for the package of interest
docs:
	cd src/$(PKGNAME); go doc -u -all > $(PKGNAME)-doc.txt

