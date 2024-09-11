export GOOS=linux && export GOARCH=amd64 && export CGO_ENABLED=0 && go build -ldflags "-s -w" -o video_server -gcflags "all=-trimpath=$GOPATH" -trimpath cmd/video_server/main.go && tar -czvf linux-amd64-video_server.tar.gz video_server
export GOOS=windows && export GOARCH=amd64 && export CGO_ENABLED=0 && go build -ldflags "-s -w" -o video_server.exe -gcflags "all=-trimpath=$GOPATH" -trimpath cmd/video_server/main.go && zip windows-video_server.zip video_server.exe
