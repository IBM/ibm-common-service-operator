KEYCLOAK_THEME_DIR ?= $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
JAR_THEME_FILE ?= cloudpak-theme.jar

default: cloudpak-theme.jar

.PHONY: cloudpak-theme.jar
cloudpak-theme.jar:
	@echo "Building the keycloak jar theme..."
	rm $(KEYCLOAK_THEME_DIR)/$(JAR_THEME_FILE) || true
	(cd $(KEYCLOAK_THEME_DIR) && zip -r ./$(JAR_THEME_FILE) META-INF theme)