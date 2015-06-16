build:
	go build discover.go
	go build explorer.go
	go build write.go

386:
	GOOS=linux GOARCH=386 go build discover.go
	GOOS=linux GOARCH=386 go build explorer.go
	GOOS=linux GOARCH=386 go build write.go

arm5:
	GOARM=5 GOOS=linux GOARCH=arm go build discover.go
	GOARM=5 GOOS=linux GOARCH=arm go build explorer.go
	GOARM=5 GOOS=linux GOARCH=arm go build write.go
