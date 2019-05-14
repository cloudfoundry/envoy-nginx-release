# envoy-nginx-release

This is a bosh release for deploying nginx in a sidecar container masquerading as envoy to the outside world. Yes, kinda hacky!

The job/package has to be `envoy_windows` because that's what diego components look for.

Expected to be consumed via `cf-deployment/operations/experimental/enable-routing-integrity-windowsXXXX.yml`.

Also see: https://github.com/cloudfoundry/envoy-release
