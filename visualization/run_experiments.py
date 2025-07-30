#!/usr/bin/env python3
# -*- coding:utf-8 -*-

"""
@author:    yanghu
@time:      2025/7/27
"""
import signal
import subprocess
import os

from build import build, SRC_MAIN


CONFIG_PATH = "../configs/config-map.yaml"
LOG_DIR = "../logs"
FILES_DIR = "../files"
VIS_DIR = "../visualization"

# 确保 logs 目录存在
os.makedirs(LOG_DIR, exist_ok=True)


def start_server(fault_number: int = 0, total_number: int = 4, pacemaker_loaded: bool = False):
    if total_number < 4 or total_number > 16:
        raise Exception("total number can't lt 4 or gt 16")

    if fault_number < 0 or fault_number > 5:
        raise Exception("fault number can't lt 0 or gt 5")

    for node_id in range(total_number):
        print(f"🚀 Starting instance with id={node_id}")

        log_file = open(f"{LOG_DIR}/main_{node_id}.log", "w")
        process = subprocess.Popen([
            SRC_MAIN, "bhs",
            f"--config={CONFIG_PATH}",
            f"--id={node_id}",
            f"--fault_number={fault_number}",
            f"--total_number={total_number}",
            f"--pacemaker_loaded={ 'true' if pacemaker_loaded else 'false' }"
        ], stdout=log_file, stderr=subprocess.STDOUT)

        # record PID
        with open(f"{LOG_DIR}/main_{node_id}.pid", "w") as pidfile:
            pidfile.write(str(process.pid))

    print("🟢 All replicas started.")


def start_client(fault_number: int = 0, total_number: int = 4, pacemaker_loaded: bool = False):

    # start client
    print("🚀 Starting client...")
    client_log = open(f"{LOG_DIR}/main_client.log", "w")
    client_process = subprocess.Popen([
        SRC_MAIN, "bhs-client",
        f"--config={CONFIG_PATH}",
        f"--fault_number={fault_number}",
        f"--total_number={total_number}",
        f"--pacemaker_loaded={ 'true' if pacemaker_loaded else 'false' }"
    ], stdout=client_log, stderr=subprocess.STDOUT)

    # 写入客户端 PID
    with open(f"{LOG_DIR}/main_client.pid", "w") as f:
        f.write(str(client_process.pid))

    return client_process


def run(fault_number: int = 0, total_number: int = 4, pacemaker_loaded: bool = False):
    start_server(fault_number, total_number, pacemaker_loaded)

    client_process = start_client(fault_number, total_number, pacemaker_loaded)

    print(f"⏳ Waiting for client (PID {client_process.pid}) to finish...")

    # 等待客户端结束
    client_process.wait()

    print("✅ Client finished. Stopping all replicas...")


def stop(total_number: int = 4):

    for node_id in range(total_number):
        pid_file = os.path.join(LOG_DIR, f"main_{node_id}.pid")
        if os.path.isfile(pid_file):
            with open(pid_file) as f:
                pid = int(f.read().strip())
            print(f"🛑 Stopping instance id={node_id} (PID {pid})...")
            try:
                os.kill(pid, signal.SIGTERM)
            except ProcessLookupError:
                print(f"⚠️  Process {pid} already terminated.")
            os.remove(pid_file)
        else:
            print(f"❓ PID file for id={node_id} not found.")

    print("✅ All server stopped.")

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


def run_experiments(scenario: str = "a"):
    build()

    def _run(_fault_number, _total_number, _pacemaker_loaded):
        try:
            run(_fault_number, _total_number, _pacemaker_loaded)
        finally:
            stop(_total_number)
            print("🛑 All replicas stopped.")

    if scenario == "a":
        fault_number = 0

        # 4 <= total_number <= 16
        for total_number in range(4, 17):

            for times in range(10):

                print(f"start scenario {scenario}: total_number {total_number} times {times + 1} pm False")

                _run(fault_number, total_number, False)

                print(f"start scenario {scenario}: total_number {total_number} times {times + 1} pm True")

                _run(fault_number, total_number, True)


    elif scenario == "b":

        # 1 <= fault_number <= 5
        for fault_number in range(4, 6):

            total_number = 3 * fault_number + 1

            for times in range(10):

                print(f"start scenario {scenario}: fault_number {fault_number} times {times + 1} pm False")

                _run(fault_number, total_number, False)

                print(f"start scenario {scenario}: fault_number {fault_number} times {times + 1} pm True")

                _run(fault_number, total_number, True)

    elif scenario == "c":

        for fault_number in range(1, 6):

            # range(4, 7)
            for total_number in range(3 * fault_number + 1, 16):

                filename_with_pm = f"../files/metric_with_total_{total_number}_fault_{fault_number}_with_pacemaker.csv"
                filename_without_pm = f"../files/metric_with_total_{total_number}_fault_{fault_number}_without_pacemaker.csv"

                if not os.path.exists(filename_without_pm):

                    for times in range(10):

                        print(f"start scenario {scenario}: fault_number {fault_number} total_number {total_number} times {times + 1} pm False")

                        _run(fault_number, total_number, False)

                if not os.path.exists(filename_with_pm):

                    for times in range(10):

                        print(f"start scenario {scenario}: fault_number {fault_number} total_number {total_number} times {times + 1} pm True")

                        _run(fault_number, total_number, True)


if __name__ == '__main__':
    build()
    # run_experiments("a")
    # run_experiments("b")
    run_experiments("c")
