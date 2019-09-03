# envoy-nginx-release

This is a bosh release for deploying nginx in a sidecar container on windows masquerading as envoy to the outside world. Yes, kinda hacky!

This enables communication between the router and the application sidecar container to be encrypted via TLS.

Expected to be used via [enable-nginx-routing-integrity-windows2019.yml](https://github.com/cloudfoundry/cf-deployment/blob/develop/operations/experimental/enable-nginx-routing-integrity-windows2019.yml).

The job/package has to be `envoy_windows` because that's what diego components look for.

Also see: https://github.com/cloudfoundry/envoy-release

### update nginx
Run `scripts/update-nginx-blob`

### update package specs
Run `scripts/sync-package-specs`
