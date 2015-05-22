GO = go
PEG = peg

.SUFFIXES: .peg .peg.go

.PHONY: all test clean
all: parse.peg.go

.peg.peg.go:
	$(PEG) -switch -inline $<

test: all
	$(GO) test ./...

clean:
	$(RM) *.peg.go
