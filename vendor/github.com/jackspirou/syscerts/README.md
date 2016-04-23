# syscerts
Gather local system certificates in Go via a public `SystemRootsPool` method.

#### What does this do?
Provide a way to gather local system certificates
on different OS platforms.

#### How does it do it?
It uses the `crypto/x509` package and provides a single public method called
`SystemRootsPool()` to return a `*x509.CertPool` object.

#### How do you use it?
```Go
// gather CA certs
certpool := syscerts.SystemRootsPool()

// place them in an HTTP client for trusted SSL/TLS connections
tlsConfig := &tls.Config{RootCAs: certpool}
transport := &http.Transport{TLSClientConfig: tlsConfig}
client := &http.Client{Transport: transport}

// make a request
resp, err := client.Do(req)
```

#### Why even do it?
The `crypto/x509` package already has a `systemRootsPool` method.
The `crypto/x509.systemRootsPool` method is almost the same as
`github.com/jackspirou/syscerts.SystemRootsPool`.
The difference? The `crypto/x509.systemRootsPool` method is private so you
cannot access it. :(

There are plans for the `crypto/x509.systemRootsPool` method to become public
in Go 1.7. When this happens you might no longer need `github.com/jackspirou/syscerts.SystemRootsPool`.

The only reason you may still use this package after the Go 1.7 release might
be for the Mac OSX System Keychain certs which are not included in the
`crypto/x509` package. Relevant lines below:

* https://github.com/jackspirou/syscerts/blob/master/root_darwin.go#L24-L32

Find more about this Go issue here: https://github.com/golang/go/issues/13335
