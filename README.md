Simple proxy written on Go Lang as pet project. Has only feature of proxying request to provided url, forwarding headers and status code of response

## How to run

```bash
    go run proxy.go
```

By default runs on port 8080. No ability to change port.

## How to use

Let's say you have a service running on `http://localhost:8080` and you want to proxy it to `https://google.com`. You can do it like this:

```bash
    curl thttp://localhost:8080/go/https://google.com
```
