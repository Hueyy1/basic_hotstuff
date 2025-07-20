#!/bin/bash

go build -o src/main src/main.go

NUM=${1:-4}

# 启动 10 个实例，id 从 0 到 9
for ((i=0; i<NUM; i++)); do
  echo "Starting instance with id=$i"
  ./src/main bhs --config=./configs/config-map.yaml --id=$i --replica_number=$NUM > "./logs/main_$i.log" 2>&1 &
  echo $! > "./logs/main_$i.pid"
done

echo "All instances started."

# 启动客户端
echo "🚀 Starting client..."
./src/main bhs-client --config=./configs/config-map.yaml --replica_number=$NUM > "./logs/main_client.log" 2>&1 &
CLIENT_PID=$!
echo $CLIENT_PID > "./logs/main_client.pid"

echo "⏳ Waiting for client (PID $CLIENT_PID) to finish..."

# 等待 client 进程结束
wait $CLIENT_PID

echo "✅ Client finished. Stopping all replicas..."
./stop.sh $NUM
