<img src="./stackstate.png" align="right" alt="StackState logo">

# Steadybit extension-stackstate

A [Steadybit](https://www.steadybit.com/) extension to integrate [Stack State](https://www.stackstate.com/) into Steadybit.

Learn about the capabilities of this extension in our [Reliability Hub](https://hub.steadybit.com/extension/com.github.steadybit.extension_stackstate).

## Configuration

| Environment Variable               | Helm value              | Meaning                                                                      | Required | Default |
|------------------------------------|-------------------------|------------------------------------------------------------------------------|----------|---------|
| `STEADYBIT_EXTENSION_API_TOKEN`    | `stackstate.apiToken`   | Stack State API Token                                                        | yes      |         |
| `STEADYBIT_EXTENSION_API_BASE_URL` | `stackstate.apiBaseUrl` | Stack State API Base URL (example: https://steadybit.app.stackstate.io/api/) | yes      |         |


The extension supports all environment variables provided by [steadybit/extension-kit](https://github.com/steadybit/extension-kit#environment-variables).

## Installation

### Using Docker

```sh
$ docker run \
  --rm \
  -p 8080 \
  --name steadybit-extension-stackstate \
  ghcr.io/steadybit/extension-stackstate:latest
```

### Using Helm in Kubernetes

```sh
$ helm repo add steadybit-extension-stackstate https://steadybit.github.io/extension-stackstate
$ helm repo update
$ helm upgrade steadybit-extension-stackstate \
    --install \
    --wait \
    --timeout 5m0s \
    --create-namespace \
    --namespace steadybit-extension \
    steadybit-extension-stackstate/steadybit-extension-stackstate
```

## Register the extension

Make sure to register the extension at the steadybit platform. Please refer to
the [documentation](https://docs.steadybit.com/integrate-with-steadybit/extensions/extension-installation) for more information.
