VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
GOFLAGS := CGO_ENABLED=0

.PHONY: scoutd test clean frontend rootfs squashfs initramfs image bundle

scoutd:
	cd scoutd && $(GOFLAGS) go build -ldflags "-X main.version=$(VERSION)" -o ../build/scoutd ./cmd/scoutd/

test:
	cd scoutd && go test ./...

clean:
	rm -f build/scoutd

# Placeholder targets — will be implemented in later months.
frontend:
	@echo "TODO: build Vue.js frontend (cd web && npm run build)"

rootfs:
	@echo "TODO: build Alpine rootfs"

squashfs:
	@echo "TODO: compress rootfs to SquashFS"

initramfs:
	@echo "TODO: build initramfs"

image:
	@echo "TODO: build full disk image"

bundle:
	@echo "TODO: build RAUC OTA bundle"
