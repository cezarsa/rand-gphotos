build-pi:
	mkdir -p build
	GOOS=linux GOARCH=arm GOARM=5 go build -o ./build/rand-gphotos-pi

.PHONY: build-pi