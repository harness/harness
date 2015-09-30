.. sentry:edition:: self

    Raven Go
    ========

.. sentry:edition:: hosted, on-premise

    .. class:: platform-go

    Go
    ==

Raven-Go provides a Sentry client implementation for the Go programming
language.

Installation
------------

Raven-Go can be installed like any other Go library through ``go get``::

    $ go get github.com/getsentry/raven-go

Minimal Example
---------------

.. sourcecode:: go

    package main

    import (
        "github.com/getsentry/raven-go"
    )

    func main() {
        raven.SetDSN("___DSN___")

        _, err := DoSomethingThatFails()
        if err != nil {
            raven.CaptureErrorAndWait(err, nil);
        }
    }
