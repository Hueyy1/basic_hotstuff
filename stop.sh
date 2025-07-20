#!/bin/bash

# 获取传入的副本数量参数（默认是 10）
NUM=${1:-4}
last=$((NUM - 1))

# 停止 id=0 到 id=9 的实例
for ((i=0; i<NUM; i++)); do
  pid_file="./logs/main_$i.pid"
  if [[ -f $pid_file ]]; then
    pid=$(cat $pid_file)
    echo "Stopping instance id=$i (PID $pid)..."
    kill $pid 2>/dev/null
    rm -f "$pid_file"
  else
    echo "PID file for id=$i not found."
  fi
done

echo "All instances stopped."

pid_file="./logs/main_client.pid"
pid=$(cat $pid_file)
kill $pid 2>/dev/null
rm -f "$pid_file"

echo "Client stopped."

cp "./files/metric_$last.csv" "./visualization/metric_$NUM.csv"

echo "CP csv"

for ((i=0; i<NUM; i++)); do
  rm "./files/metric_$i.csv"
done
