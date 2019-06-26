# envoy-nginx-release

This is a bosh release for deploying nginx in a sidecar container masquerading as envoy to the outside world. Yes, kinda hacky!

The job/package has to be `envoy_windows` because that's what diego components look for.

Expected to be consumed via `cf-deployment/operations/experimental/enable-routing-integrity-windowsXXXX.yml`.

Also see: https://github.com/cloudfoundry/envoy-release


## Steps to update `nginx` in the bosh release

Download nginx
```
cd /tmp
curl http://nginx.org/download/nginx-1.17.1.zip --output nginx.zip
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
cd ~/workspace/envoy-nginx-release
bosh add-blob /tmp/nginx-stuff/envoy-nginx.zip envoy-nginx/envoy-nginx.zip
set_bosh_windows_s3_blobstore
bosh upload-blobs
```
