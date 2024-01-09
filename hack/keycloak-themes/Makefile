KEYCLOAK_THEME_DIR ?= hack/keycloak-themes
JAR_THEME_FILE ?= cloudpak-theme.jar

default: cloudpak-theme.jar

cloudpak-theme.jar:
	@echo "Building the keycloak jar theme..."
	@(cd $(KEYCLOAK_THEME_DIR) && zip -r ./$(JAR_THEME_FILE) META-INF theme)