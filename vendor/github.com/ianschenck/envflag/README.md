envflag
=======

Golang flags, but bolted onto the environment rather than the command-line.

Read the [godocs](http://godoc.org/github.com/ianschenck/envflag).

Motivation
==========

Some like the distinction that command-line flags control behavior
while environment variables configure. Also
[12-factor](http://12factor.net/) recommends the use of environment
variables for configuration. The interface of the golang flag package
is well designed and easy to use, and allows for other lists
(os.Environ() vs os.Args) to be parsed as flags. It makes sense then
to use the same interface, the same types, and the same parsing
(caveat: there is some ugly string hacking to make environment
variables look like flags) to the same ends.

Differences
===========

Calling `flag.Parse()` will not parse environment flags. Calling
`envflag.Parse()` will not parse command-line flags. There is no good
reason to combine these two when the net savings is a single line in a
`func main()`. Furthermore, doing so would require users to accept a
precedence order of my choosing.

The presence of an environment variable named `h` or `help` will
probably cause problems (print Usage and os.Exit(0)). Work around this
by defining those flags somewhere (and ignoring them).

Before calling `Flagset.Parse` on `EnvironmentFlags`, the environment
variables being passed to `Parse` are trimmed down using
`Lookup`. This behavior is different from `flag.Parse` in that extra
environment variables are ignored (and won't crash `envflag.Parse`).
