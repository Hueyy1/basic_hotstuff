#!/usr/bin/env python3
# -*- coding:utf-8 -*-

"""
@author:    yanghu
@time:      2025/7/27
"""
import signal
#!/usr/bin/env python3

import subprocess
import os
import sys
from time import sleep

SRC_MAIN = "../src/main"
CONFIG_PATH = "../configs/config-map.yaml"
LOG_DIR = "../logs"
FILES_DIR = "../files"
VIS_DIR = "../visualization"


def build():
    # 编译 Go 程序
    print("🔨 Building Go program...")
    build_cmd = ["go", "build", "-o", SRC_MAIN, "../src/main.go"]
    result = subprocess.run(build_cmd)
    if result.returncode != 0:
        print("❌ Go build failed.")
        sys.exit(1)
    print("✅ Build complete.")


def run(FaultNum: int = 0):

    # 确保 logs 目录存在
    os.makedirs(LOG_DIR, exist_ok=True)

    if FaultNum == 0:
        # start 4 nodes, without Fault
        for i in range(4):
            print(f"🚀 Starting instance with id={i}")
            log_file = open(f"{LOG_DIR}/main_{i}.log", "w")
            process = subprocess.Popen([
                SRC_MAIN, "bhs",
                f"--config={CONFIG_PATH}",
                f"--id={i}",
                f"--fault_number=0",
            ], stdout=log_file, stderr=subprocess.STDOUT)
            # 记录 PID
            with open(f"{LOG_DIR}/main_{i}.pid", "w") as pidfile:
                pidfile.write(str(process.pid))

    else:
        # start 3*F+1 nodes, with f fault nodes

        for i in range(3 * FaultNum + 1):
            print(f"🚀 Starting instance with id={i}")
            log_file = open(f"{LOG_DIR}/main_{i}.log", "w")
            process = subprocess.Popen([
                SRC_MAIN, "bhs",
                f"--config={CONFIG_PATH}",
                f"--id={i}",
                f"--fault_number={FaultNum}",
            ], stdout=log_file, stderr=subprocess.STDOUT)
            # 记录 PID
            with open(f"{LOG_DIR}/main_{i}.pid", "w") as pidfile:
                pidfile.write(str(process.pid))

    print("🟢 All replicas started.")

    # start client
    print("🚀 Starting client...")
    client_log = open(f"{LOG_DIR}/main_client.log", "w")
    client_process = subprocess.Popen([
        SRC_MAIN, "bhs-client",
        f"--config={CONFIG_PATH}",
        f"--fault_number={FaultNum}"
    ], stdout=client_log, stderr=subprocess.STDOUT)

    # 写入客户端 PID
    with open(f"{LOG_DIR}/main_client.pid", "w") as f:
        f.write(str(client_process.pid))

    print(f"⏳ Waiting for client (PID {client_process.pid}) to finish...")

    # 等待客户端结束
    client_process.wait()

    print("✅ Client finished. Stopping all replicas...")

    # 调用 stop.sh 脚本
    stop(FaultNum)

    # print("🛑 All replicas stopped.")


def stop(FaultNum: int = 0):

    total_size = 3 * FaultNum + 1 if FaultNum != 0 else 4

    # 停止副本实例
    for i in range(total_size):
        pid_file = os.path.join(LOG_DIR, f"main_{i}.pid")
        if os.path.isfile(pid_file):
            with open(pid_file) as f:
                pid = int(f.read().strip())
            print(f"🛑 Stopping instance id={i} (PID {pid})...")
            try:
                os.kill(pid, signal.SIGTERM)
            except ProcessLookupError:
                print(f"⚠️  Process {pid} already terminated.")
            os.remove(pid_file)
        else:
            print(f"❓ PID file for id={i} not found.")

    print("✅ All instances stopped.")

    # 停止客户端
    client_pid_file = os.path.join(LOG_DIR, "main_client.pid")
    if os.path.isfile(client_pid_file):
        with open(client_pid_file) as f:
            client_pid = int(f.read().strip())
        print(f"🛑 Stopping client (PID {client_pid})...")
        try:
            os.kill(client_pid, signal.SIGTERM)
        except ProcessLookupError:
            print("⚠️  Client process already terminated.")
        os.remove(client_pid_file)
        print("✅ Client stopped.")
    else:
        print("❓ Client PID file not found.")

    # 拷贝最终 metric CSV 文件
    # src_csv = os.path.join(FILES_DIR, f"metric_with_{FaultNum}_fault.csv")

    # dst_csv = os.path.join(VIS_DIR, f"metric_{NUM}.csv")
    #
    # try:
    #     shutil.copy(src_csv, dst_csv)
    #     print(f"📁 Copied {src_csv} → {dst_csv}")
    # except Exception as e:
    #     print(f"❌ Failed to copy metric CSV: {e}")
    #
    # # 删除所有临时 metric CSV 文件
    # for i in range(NUM):
    #     try:
    #         os.remove(os.path.join(FILES_DIR, f"metric_{i}.csv"))
    #     except FileNotFoundError:
    #         pass
    #
    # print("🧹 Cleaned up metric files.")


if __name__ == '__main__':
    # build()

    # FaultNum = int(sys.argv[1]) if len(sys.argv) > 1 else 0

    # for FaultNum in range(5, 6):
    #
    #     for i in range(3):
    #         print(f"start fault {FaultNum} {i + 1} times")
    #         run(FaultNum)

    stop(5)

    # run(3)
