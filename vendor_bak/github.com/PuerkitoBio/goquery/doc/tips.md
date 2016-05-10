# Tips and tricks

## Handle Non-UTF8 html Pages

The `go.net/html` package used by `goquery` requires that the html document is UTF-8 encoded. When you know the encoding of the html page is not UTF-8, you can use the `iconv` package to convert it to UTF-8 (there are various implementation of the `iconv` API, see [godoc.org][iconv] for other options):

```
$ go get -u github.com/djimenez/iconv-go
```

and then:

```
// Load the URL
res, err := http.Get(url)
if e != nil {
    // handle error
}
defer res.Body.Close()

// Convert the designated charset HTML to utf-8 encoded HTML.
// `charset` being one of the charsets known by the iconv package.
utfBody, err := iconv.NewReader(res.Body, charset, "utf-8")
if err != nil {
    // handler error
}

// use utfBody using goquery
doc, err := goquery.NewDocumentFromReader(utfBody)
if err != nil {
    // handler error
}
// use doc...
```

Thanks to github user @YuheiNakasaka.

Actually, the official go.text repository covers this use case too, see its [godoc page][text] for the details.


## Handle Javascript-based Pages

`goquery` is great to handle normal html pages, but when most of the page is build dynamically using javascript, there's not much it can do. There are various options when faced with this problem:

* Use a headless browser such as [webloop][].
* Use a Go javascript parser package, such as [otto][].

You can find a code example using `otto` [in this gist][exotto]. Thanks to github user @cryptix.

## For Loop

If all you need is a normal `for` loop over all nodes in the current selection, where `Map/Each`-style iteration is not necessary, you can use the following:

```
sel := Doc().Find(".selector")
for i := range sel.Nodes {
	single := sel.Eq(i)
    // use `single` as a selection of 1 node
}
```

Thanks to github user @jmoiron.

[webloop]: https://github.com/sourcegraph/webloop
[otto]: https://github.com/robertkrimen/otto
[exotto]: https://gist.github.com/cryptix/87127f76a94183747b53
[iconv]: http://godoc.org/?q=iconv
[text]: http://godoc.org/code.google.com/p/go.text/encoding
