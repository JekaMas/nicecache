get-glide:
	@echo "Installing glide ..."
	go get -u github.com/Masterminds/glide

deps:
	@echo "Deps..."
	go glide install --force

benchmark:
	go test -bench=. -benchmem -benchtime=20s -timeout=6000000s -v

test:
	go test -timeout=180s -v

benchmark-race:
	go test -race -bench=. -benchmem -benchtime=20s -timeout=6000000s -v

.PHONY: benchmark-race benchmark test deps get-glide