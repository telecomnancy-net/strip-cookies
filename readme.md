# strip-cookies

Traefik middleware plugin to strip request cookies by name.

Based on [traefik/plugindemo](https://github.com/traefik/plugindemo).

## Configuration

### Static

```yaml
experimental:
  plugins:
    strip-cookies:
      moduleName: "github.com/telecomnancy-net/strip-cookies"
      version: "v0.1.0"
```

### Dynamic

```yaml
http:
  middlewares:
    strip-tracking:
      plugin:
        strip-cookies:
          cookies:
            - tracking_id
            - session_id
```

### Local mode

```
./plugins-local/src/github.com/telecomnancy-net/strip-cookies/
```

```yaml
experimental:
  localPlugins:
    strip-cookies:
      moduleName: "github.com/telecomnancy-net/strip-cookies"
```

