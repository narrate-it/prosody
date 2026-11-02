FLUTTER ?= flutter
DEVICE ?=

.PHONY: preview
preview:
	cd mobile && "$(FLUTTER)" run --debug -t lib/preview_main.dart $(if $(DEVICE),-d "$(DEVICE)")
