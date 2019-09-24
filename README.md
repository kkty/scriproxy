# Scriproxy

[![codecov](https://codecov.io/gh/kkty/scriproxy/branch/master/graph/badge.svg)](https://codecov.io/gh/kkty/scriproxy)

## Features

- Scriproxy is an easy-to-use **scriptable** reverse proxy.
- You can write a script to select the upstream server dynamically for each request. Header values, query values, and request paths can be used.
- You can do more than dynamically selecting the upstream server. It is also possible to rewrite header values, query values, and others.
- [Tengo language](https://github.com/d5/tengo) is used for writing scripts. It has go-like syntax and good standard libraries.

## Install

```console
$ go install github.com/kkty/scriproxy
```

## Usage

```console
$ scriproxy --help
Usage:
  scriproxy [OPTIONS]

Application Options:
      --script=    The path to a tengo script for rewriting requests
      --libraries= The tengo libraries used in the script, separated by commas

Help Options:
  -h, --help       Show this help message
```

You can write a go-like script for rewriting HTTP requests. Request paths, header values, query values, and others can be used (and can be modified as well.) The example scripts are shown in the "Example Scripts" section below.

[Tengo language](https://github.com/d5/tengo) is used as a backend. It has many cool [built-in standard libraries](https://github.com/d5/tengo/blob/master/docs/stdlib.md), and they can be used as well. To use the standard libraries, the library names should be specified in the command line arguments. For example, if you want to use `text` and `fmt`, `--libraries=text,fmt` should be added to the command line arguments.

An example command to start a proxy server is as follows.

```console
$ scriproxy --script /path/to/script.go --libraries fmt,text
```

## Example Scripts

These example scripts may give you the idea of what is possible with Scriproxy. **You can combine the below scripts and do a lot more!**

For the list of available values and functions, refer to the "Notes" section.

### Simple proxy

```go
// req.url corresponds to the url to which the request is sent.
req.url.scheme = "https"
req.url.host = "example.com"

// req.host represents the host header value
// You (almost always) have to set this value
// In most cases, req.host should be the same value as req.url.host
req.host = "example.com"
```

With this script, all the requests to the proxy server are routed to `https://example.com` with the host header value of `example.com`. The header/query values and the request path will be kept unchanged.

### Using query values

Query values can be used for selecting the upstream server.

```go
// Retrive values from the query
req.url.host = req.url.query.get("host")
req.url.scheme = req.url.query.get("scheme")

// Remove the query values to tidy up
req.url.query.del("host")
req.url.query.del("scheme")

// This is (almost always) neccessary!
req.host = req.url.host
```

With this script, requests to `/foo?host=example.com&scheme=http` are routed to `http://example.com/foo` and requests to `/foo?host=example.org&scheme=https` are routed to `https://example.org/foo`.

### Using host header values

You can use the host header value for selecting to which upstream server to connect.

```go
// Note that you shoud specify `--libraries=text` in the command line arguments!
text := import("text")

// We are expecting req.host to be "example.com.local", "example.com.secure.local", or something like that

splitted := text.split(req.host, ".")
l := len(splitted)

if splitted[l-2] == "secure" {
  req.url.scheme = "https"
  req.url.host = text.join(splitted[:l-2], ".")
} else {
  req.url.scheme = "http"
  req.url.host = text.join(splitted[:l-1], ".")
}

req.host = req.url.host
```

With this script, requests whose host header values are set to `example.com.secure.local` are routed to `https://example.com`, and requests with `example.com.local` host header values are routed to `http://example.com`.

### Using header values

You can use header values to select the upstream server.

```go
// Note that you shoud specify `--libraries=text` in the command line arguments!
text := import("text")

// You can use "User-Agent" instead of "user-agent"
// It acts like https://golang.org/pkg/net/http/#Header.Get
ua := req.header.get("user-agent")
ua = text.to_lower(ua)

if text.contains(ua, "iphone") {
  req.url.host = "example.com"
} else {
  req.url.host = "example.org"
}

// Overwriting the user agent header value
req.header.set("user-agent", "my-proxy")

req.url.scheme = "https"
req.host = req.url.host
```

With this script, requests from iPhones are routed to `https://example.org` and the other requests are routed to `https://example.com`.

`req.header.set("host", "...")` does not work. To rewrite host header values, you should modify `req.host` instead. This is in line with the behavior of Go's [http.Request](https://golang.org/pkg/net/http/#Request).

## Notes

- `req.host`, `req.url.scheme`, `req.url.host` and `req.url.path` can be used as values and can be modified.
- `req.url.query.get("key")`, `req.url.query.set("key", "value")` and `req.url.query.del("key")` functions can be used to rewrite query values.
- `req.header.get("key")`, `req.header.set("key", "value")`, `req.header.add("key", "value")` and `req.header.del("key")` functions can be used to rewrite header values.
- The behaviors of the values and the functions above are simillar to Go's [http.Request](https://golang.org/pkg/net/http/#Request).
- Scriproxy can serve around a few thousand requests per second on a modern 2-core machine.
- By default, port 80 is used for listening. You can change it by setting `PORT` environment variable.

## Logging

Scriproxy has built-in logging. The example log is as follows. Note that, in production, one log entry is printed without line breaks.

```json
{
  "level": "info",
  "ts": 1568759341.6854603,
  "caller": "scriproxy/server.go:101",
  "msg": "received_response",
  "method": "GET",
  "remote_addr": "127.0.0.1:40342",
  "original_user_agent": "curl/7.58.0",
  "original_host": "localhost:8080",
  "original_url_path": "/foo",
  "original_url_query": "host=example.com&scheme=https",
  "host": "example.com",
  "url_scheme": "https",
  "url_host": "example.com",
  "url_path": "/foo",
  "url_query": "",
  "status_code": 404,
  "elapsed": 0.109501097
}
```

The above log was obtained by the following commands.

```console
$ cat > /tmp/script <<EOF
req.url.host = req.url.query.get("host")
req.url.scheme = req.url.query.get("scheme")
req.url.query.del("host")
req.url.query.del("scheme")
req.host = req.url.host
EOF
$ PORT=8080 scriproxy --script=/tmp/script
```

```console
$ curl 'http://localhost:8080/foo?host=example.com&scheme=https'
```
