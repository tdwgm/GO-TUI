.PHONY: all build test run clean

all: build

build:
	$(MAKE) -C tui-example build

test:
	$(MAKE) -C tui-example test

run: build
	./tui-example/bin/tui-example

clean:
	$(MAKE) -C tui-example clean
