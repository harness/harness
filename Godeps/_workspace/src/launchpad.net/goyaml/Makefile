include $(GOROOT)/src/Make.inc

YAML=yaml-0.1.3
LIBYAML=$(PWD)/$(YAML)/src/.libs/libyaml.a

TARG=launchpad.net/goyaml

GOFILES=\
	goyaml.go\
	resolve.go\

CGOFILES=\
	decode.go\
	encode.go\

CGO_OFILES+=\
	helpers.o\
	api.o\
	scanner.o\
	reader.o\
	parser.o\
	writer.o\
	emitter.o\

GOFMT=gofmt

BADFMT:=$(shell $(GOFMT) -l $(GOFILES) $(CGOFILES) $(wildcard *_test.go))

all: package
gofmt: $(BADFMT)
	@for F in $(BADFMT); do $(GOFMT) -w $$F && echo $$F; done

include $(GOROOT)/src/Make.pkg

ifneq ($(BADFMT),)
ifneq ($(MAKECMDGOALS),gofmt)
$(warning WARNING: make gofmt: $(BADFMT))
endif
endif
