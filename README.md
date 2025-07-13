# HotStuff

## Run by source code
```shell
cd src
go build -o main
./main generate-certs --config=../configs/config-map.yaml
./main bhs --config=../configs/config-map.yaml --id=0
./main bhs --config=../configs/config-map.yaml --id=1
./main bhs --config=../configs/config-map.yaml --id=2
./main bhs --config=../configs/config-map.yaml --id=3
```

## Gorums Proto
```shell
# make sure install github.com/relab/gorums v0.7.1-0.20220818130557-8533cb369cd6
# the lastest version has problem with protoc cmd
# solution: comment the line in basichotstuff_gorums.pb.go after compile proto file >> "_ = gorums.EnforceVersion(gorums.MaxVersion - 9)"

protoc -I=$(go list -m -f {{.Dir}} github.com/relab/gorums):.\
  --go_out=paths=source_relative:. \
  --gorums_out=paths=source_relative:. \
  proto/commonpb/common.proto

protoc -I=$(go list -m -f {{.Dir}} github.com/relab/gorums):.\
  --go_out=paths=source_relative:. \
  --gorums_out=paths=source_relative:. \
  proto/basichotstuffpb/basichotstuff.proto
  
protoc -I=$(go list -m -f {{.Dir}} github.com/relab/gorums):.\
  --go_out=paths=source_relative:. \
  --gorums_out=paths=source_relative:. \
  proto/clientpb/client.proto
```
