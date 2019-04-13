terraform-state-server: main.go
	go build

clean:
	rm -f terraform-state-server
