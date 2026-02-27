run:
	go run $(shell ls *.go | grep -v _test.go)
