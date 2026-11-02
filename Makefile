FLUTTER ?= flutter
DEVICE ?=

.PHONY: preview
preview:
	cd mobile && "$(FLUTTER)" run --debug $(if $(DEVICE),-d "$(DEVICE)")
