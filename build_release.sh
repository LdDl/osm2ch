# Linux build
export GOOS=linux && export GOARCH=amd64 && export CGO_ENABLED=0 && go build -ldflags "-s -w" -o ./cmd/osm2ch/osm2ch -gcflags "all=-trimpath=$GOPATH" -trimpath ./cmd/osm2ch/main.go
cd ./cmd/osm2ch && tar -czf osm2ch.tar.gz osm2ch && cd -
# Windows build
export GOOS=windows && export GOARCH=amd64 && export CGO_ENABLED=0 && go build -ldflags "-s -w" -o ./cmd/osm2ch/osm2ch.exe -gcflags "all=-trimpath=$GOPATH" -trimpath ./cmd/osm2ch/main.go
sudo apt install zip
cd ./cmd/osm2ch && zip osm2ch.zip osm2ch.exe && cd -