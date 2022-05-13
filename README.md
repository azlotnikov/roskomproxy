# RoskomProxy
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/azlotnikov/roskomproxy)
---
RoskomProxy is an example of local HTTP proxy server for bypassing blocking of sites in Russia.
Works only for blocked by domain name (such as instagram.com, facebook.com, meduza.io).

The main idea is to use custom DNS server and remove `server_name` (SNI) TLS extension from TLS client hello.

# Usage
## Launch
* Build project `go build -o roskomproxy`
* Launch with Google DNS `./roskomproxy --cert server.crt --key server.key --dns 8.8.8.8 --port 8080`

## Curl
* `curl -k -vvv https://www.instagram.com/ -x 127.0.0.1:8080 --compressed`

## macOS Safari
* Install `server.crt` (or generate your own crt/key) to Keychain
* Set `Always trust` for this certificate
* In the internet connection settings turn on HTTPS proxy `127.0.0.1`, `8080`
* Open Safari and try to open `https://facebook.com/`
