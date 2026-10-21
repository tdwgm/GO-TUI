.PHONY: all build test run clean

all: build

build:
	$(MAKE) -C tui-example build

test:
	$(MAKE) -C tui-example test
	bash bash/tests/test.sh
	(cd python && uv run pytest tests/test_model.py 2>/dev/null || true)

run: build
	./tui-example/bin/tui-example

clean:
	$(MAKE) -C tui-example clean
