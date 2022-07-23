build:
	go run .

run: build
	./server

watch:
	clear
	ulimit -n 1000
	reflex -s -r '\.go$$' make run