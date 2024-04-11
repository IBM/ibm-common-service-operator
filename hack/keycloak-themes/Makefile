KEYCLOAK_THEME_DIR ?= $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
JAR_THEME_FILE ?= cloudpak-theme.jar

default: cloudpak-theme.jar

.PHONY: cloudpak-theme.jar
cloudpak-theme.jar:
	@echo "Building the keycloak jar theme..."
	rm $(KEYCLOAK_THEME_DIR)/$(JAR_THEME_FILE) || true
	(cd $(KEYCLOAK_THEME_DIR) && zip -r ./$(JAR_THEME_FILE) META-INF theme)

cloudpak-theme-version:
	$(eval THEME_VERSION := $(shell ls $(KEYCLOAK_THEME_DIR)/theme/cloudpak/login/resources/css/ | grep .css))
	@echo "Updating the keycloak jar theme version to $(THEME_VERSION)"
	$(YQ) eval -i '.spec.template.metadata.annotations.cloudPakThemesVersion = "$(THEME_VERSION)"' $(KEYCLOAK_THEME_DIR)/../../testdata/deploy/deploy.yaml
	$(YQ) eval -i '.metadata.annotations.cloudPakThemesVersion = "$(THEME_VERSION)"' $(KEYCLOAK_THEME_DIR)/../../config/manifests/bases/ibm-common-service-operator.clusterserviceversion.yaml 
	$(YQ) eval -i '.metadata.annotations.cloudPakThemesVersion = "$(THEME_VERSION)"' $(KEYCLOAK_THEME_DIR)/../../bundle/manifests/ibm-common-service-operator.clusterserviceversion.yaml