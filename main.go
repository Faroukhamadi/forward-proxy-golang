package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

// Hop-by-hop headers. These are removed by proxies, I think it's because they're only relevant to intermediate nodes.
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

func (p Proxy) copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func (p Proxy) deleteHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

// X-Forwarded-For: client, proxy. This is a standard header that is used to track the path of a request.
// according to Google, it helps us identify the original client IP address.
func (p Proxy) appendProxyToXForwardedFor(header http.Header, host string) {
	// if it already exists, append the proxy to the list
	if prior, ok := header["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + p.Name()
	}
	// append anyways, whether it exists or not
	header.Set("X-Forwarded-For", host)
}

// implement the http.Handler interface so we can use it as a handler by making a method called ServeHTTP
func (p Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// this doesn't work in localhost, but it does work in production
	// I think so set it manually for now
	r.URL.Scheme = "http"
	if r.URL.Scheme != "http" && r.URL.Scheme != "https" {
		// if not set, set depending on tls var
		http.Error(w, "invalid scheme", http.StatusBadRequest)
		return
	}

	client := &http.Client{}

  // do this unless it panics because request can't have requestURI according to golang http docs
  r.RequestURI = ""

	p.deleteHopHeaders(r.Header)

	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		p.appendProxyToXForwardedFor(r.Header, clientIP)
	}

	resp, err := client.Do(r)

	if err != nil {
		http.Error(w, err.Error()+"fuckkkk", http.StatusInternalServerError)
	}
  // nill pointer dereference here
	defer resp.Body.Close()
  

	p.deleteHopHeaders(resp.Header)

	p.copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

}

type Filter interface {
	Name() string
}

type BeforeFilter interface {
	Filter
	ForbiddenHeaders() []string
	DoBefore(context context.Context) error // maybe a custom error but who cares, right?
}

type AfterFilter interface {
	Filter
	DoAfter(context context.Context) error // maybe a custom error but who cares, right?
}

type Proxy struct {
}

func (proxy Proxy) Name() string {
	return "cool proxy"
}

func (proxy Proxy) ForbiddenHeaders() []string {
	return []string{"header1", "head"}
}

func (proxy Proxy) DoBefore(context context.Context) error {
	return context.Err()
}

func (proxy Proxy) DoAfter(context context.Context) error {
	return context.Err()
}


func main() {
	// I can fill this up using cli flags :)
	addr1 := flag.String("addr1", ":8080", "address1 to listen on")
	// addr2 := flag.String("addr2", ":8081", "address2 to listen on")

	flag.Parse()

	handler := Proxy{}

  log.Println("starting server")
	if err := http.ListenAndServe(*addr1, handler); err != nil {
		panic(err.Error() + " We're doomed!")
	}

	// use this if it can be both, concurrency stuff very cool
	// go func() {
	// 	if err := http.ListenAndServeTLS(*addr1, "host.cert", "host.key", handler); err != nil {
	// 		panic(err.Error() + " We're doomed!")
	// 	}
	// }()

	// if err := http.ListenAndServe(*addr2, handler); err != nil {
	// 	panic(err.Error() + " We're doomed! More doomed here if you we're not using TLS!")
	// }

}
