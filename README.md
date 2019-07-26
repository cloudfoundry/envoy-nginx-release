# envoy-nginx-release

This is a bosh release for deploying nginx in a sidecar container on windows masquerading as envoy to the outside world. Yes, kinda hacky!

This enables communication between the router and the application sidecar container to be encrypted via TLS.

Expected to be used via [enable-nginx-routing-integrity-windows2019.yml](https://github.com/cloudfoundry/cf-deployment/blob/develop/operations/experimental/enable-nginx-routing-integrity-windows2019.yml).

The job/package has to be `envoy_windows` because that's what diego components look for.

Also see: https://github.com/cloudfoundry/envoy-release


## Steps to update `nginx` in the bosh release

Download nginx
```
curl http://nginx.org/download/nginx-1.17.1.zip --output /tmp/nginx.zip
```


Rezip nginx directory so its not nested
```
mkdir nginx-stuff
cd /tmp/nginx-stuff`
mv /tmp/nginx.zip .
unzip nginx.zip
cd nginx-1.17.1
zip ../envoy-nginx.zip *
```

Add and upload the envoy-nginx blob
```
cd path/to/envoy-nginx-release
bosh add-blob /tmp/nginx-stuff/envoy-nginx.zip envoy-nginx/envoy-nginx.zip
# set blobstore credentials 
set_bosh_windows_s3_blobstore
bosh upload-blobs
```
