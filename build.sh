

#GOOS=linux GOARCH=amd64 go build -o yaopool-ssl
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w"  -a -o v2ray-process -v ./main/main.go

upx -9 ./v2ray-process

CGO_ENABLED=0  GOOS=windows GOARCH=amd64 go build -ldflags="-s -w"  -o v2ray-process.exe -v ./main/main.go
upx -9 ./v2ray-process.exe

CGO_ENABLED=0  GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w"  -o v2ray-process-mac -v ./main/main.go
