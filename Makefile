build:
	go build -o odc

windows: 
	GOOS=windows GOARCH=amd64 go build -o odc.exe

clean:
	rm odc odc.exe