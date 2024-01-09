# keycloak-themes

A Keycloak theme for IBM Cloud Paks.

See https://www.keycloak.org/docs/latest/server_development/index.html#_themes

## Testing

Easiest way to play with it is to mount it into a local keycloak dev container.

1. Run the container:

   ```
   podman run -e KEYCLOAK_ADMIN=admin -e KEYCLOAK_ADMIN_PASSWORD=chooseapassword -p 8080:8080 -p 8443:8443 -v $(pwd)/theme/:/opt/keycloak/themes:Z --name rhbk registry.redhat.io/rhbk/keycloak-rhel9@sha256:d18adf0219a17b6619ddfb86a7d569019481f0315d94917793038ba5c6dc9567 start-dev
   ```

1. Log into Keycloak with your chosen admin password at http://localhost:8080/

1. Create a realm

1. Assign the theme to the realm

Any changes you make will be immediately visible, although you might need to force refresh to avoid caching.

## Developing

The easiest way to understand what you can override in the theme is to look at the existing themes. To do this, get the theme jar (something like `org.keycloak.keycloak-themes-22.0.7.redhat-00001.jar` and `org.keycloak.keycloak-admin-ui-22.0.7.redhat-00001.jar`) from an RHBK instance and extract it.

In the `theme.properties` file, you can specify a parent theme (where resources not included in your theme will be taken from) and one or more imports (additional files you want to be considered part of your theme). You can also add additinal styles and set properties for class names etc - see the RHBK themes for examples.

Any file you include in your theme will override a parent theme file, so by looking at the RHBK theme files and putting equivalents into the cloud pak theme, you can override resources.
