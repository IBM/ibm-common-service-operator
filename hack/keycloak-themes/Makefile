KEYCLOAK_THEME_DIR ?= $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
JAR_THEME_FILE ?= cloudpak-theme.jar

default: cloudpak-theme.jar

.PHONY: cloudpak-theme.jar
cloudpak-theme.jar:
	@echo "Building the keycloak jar theme..."
	rm ./$(JAR_THEME_FILE)
	(cd $(KEYCLOAK_THEME_DIR) && zip -r ./$(JAR_THEME_FILE) META-INF theme)