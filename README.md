# Scriproxy

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

### Query-based proxy

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

### Host-based proxy

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

---

**You can combine the above scripts and do a lot more!**

## Notes

- `req.host`, `req.url.scheme`, `req.url.host` and `req.url.path` can be used as values and can be modified.
- `req.url.query.get("key")`, `req.url.query.set("key", "value")` and `req.url.query.del("key")` functions can be used to rewrite query values.
- `req.header.get("key")`, `req.header.set("key", "value")`, `req.header.add("key", "value")` and `req.header.del("key")` functions can be used to rewrite header values.s
- The behaviors of the values and the functions above are simillar to Go's [http.Request](https://golang.org/pkg/net/http/#Request).
