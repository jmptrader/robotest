TARGETS := e2e suite
DOCKER_REPO := quay.io/gravitational
VERSION ?= $(shell git describe --long --tags --always|awk -F'[.-]' '{print $$1 "." $$2 "." $$4}')
LATEST := latest

TERRAFORM_VERSION := 0.9.5
CHROMEDRIVER_VERSION := 2.29
BUILD_ARGS := --build-arg TERRAFORM_VERSION=$(TERRAFORM_VERSION) --build-arg CHROMEDRIVER_VERSION=$(CHROMEDRIVER_VERSION)

.PHONY: all
all: $(TARGETS)

BINARIES := $(addprefix ../build/robotest-,$(TARGETS))

$(BINARIES):
	cd .. && make build

.PHONY: $(TARGETS)
$(TARGETS): $(BINARIES)
	$(eval TEMPDIR = "$(shell mktemp -d)")
	$(eval IMAGE = $(DOCKER_REPO)/robotest-$@:$(VERSION))
	if [ -z "$(TEMPDIR)" ]; then \
	  echo "TEMPDIR is not set"; exit 1; \
	fi;
	mkdir -p $(TEMPDIR)/build
	cp -a ../build/robotest-$@ $(TEMPDIR)/build/
	cp -r $@/* $(TEMPDIR)/
	cd $(TEMPDIR) && docker build $(BUILD_ARGS) --rm=true --pull -t $(IMAGE) .
	rm -rf $(TEMPDIR)
	@echo Built $(IMAGE)

# Publish 
 
DOCKER_IMG = $(addprefix $(DOCKER_REPO)/robotest-,$(TARGETS))

.PHONY: publish
publish: $(DOCKER_IMG)

.PHONY: $(DOCKER_IMG)
$(DOCKER_IMG): $(TARGETS)
	docker tag $@:$(VERSION) $@:$(LATEST)
	docker push $@:$(VERSION)
	docker push $@:$(LATEST)

.PHONY: what-version
what-version:
	@echo $(VERSION)