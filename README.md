# HotStuff

## build locally
```shell
cd src
go build -o main
```

## Command Line Usage
```shell
./main --help

    Usage:
       [flags]
       [command]
    
    Available Commands:
      bhs            
      bhs-client     
      completion     Generate the autocompletion script for the specified shell
      generate-certs 
      help           Help about any command
      test-conf      
    
    Flags:
      -h, --help            help for this command
    
    Use " [command] --help" for more information about a command.


./main bhs --help

    basic hotstuff
    
    Usage:
       bhs [flags]
    
    Flags:
      -f, --fault_number int   fault_number, start from 0 to 5
      -h, --help               help for bhs
      -i, --id int             id of the replica, start from 0
      -p, --pacemaker_loaded   pacemaker_loaded, true or false, default false
      -t, --total_number int   total_number, start from 4 to 16 (default 4)
```

## Getting Started
```shell
# generate certs
./main generate-certs

# start nodes
./main bhs --id=0 --fault_number=0 --total_number=4 --pacemaker_loaded=false
./main bhs --id=1 --fault_number=0 --total_number=4 --pacemaker_loaded=false
./main bhs --id=2 --fault_number=0 --total_number=4 --pacemaker_loaded=false
./main bhs --id=3 --fault_number=0 --total_number=4 --pacemaker_loaded=false

# start client
bhs-client --fault_number=0 --total_number=4 --pacemaker_loaded=false
```

## Visualization
```shell
# create virtual environment, python >= 3.11
cd visualization
python3 -m venv venv
pip install -r requirements.txt

# run nodes and client, to get raw metrics data
python3 run_experiments.py

# generate figures
python3 generate_figures.py
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
