#!/usr/bin/env python3
# -*- coding:utf-8 -*-

"""
@author:    yanghu
@time:      2025/7/20
"""

import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import glob
import re
from datetime import timedelta


def run():
    faults = []
    avg_latencies = []
    throughputs = []

    # 匹配所有数据文件
    csv_files = glob.glob("../files/metric_with_*_fault.csv")
    csv_files.sort()
    for file in csv_files:
        match = re.search(r"(\d+)", file)
        fault_number = int(match.group(1))

        df = pd.read_csv(file, parse_dates=["payload", "latency_ms"])
        df["latency_ms"] = pd.to_numeric(df["latency_ms"], errors="coerce")
        df["payload"] = pd.to_numeric(df["payload"], errors="coerce")
        df = df.dropna(subset=["latency_ms", "payload"])

        # latency_ms sort
        df_sorted = df.sort_values(by="latency_ms")

        # remove max 10% and min 10%
        total_len = len(df_sorted)
        cut = int(total_len * 0.10)
        df_trimmed = df_sorted.iloc[cut:total_len - cut]  # 中间 80%

        # 计算平均延迟
        avg_latency = df_trimmed["latency_ms"].mean()

        # 吞吐量计算（单位：payload/ms → 乘 1000 得到 payload/sec）
        total_payload_byte = df["payload"].sum()
        total_latency_ms = df["latency_ms"].sum()
        throughput = (total_payload_byte / total_latency_ms) * 1000

        # 收集结果
        faults.append(fault_number)
        avg_latencies.append(avg_latency)
        throughputs.append(throughput)

    # 按 fault_number 升序排序（确保图像有序）
    sorted_data = sorted(zip(faults, avg_latencies, throughputs))
    faults, avg_latencies, throughputs = zip(*sorted_data)

    # 绘图
    plt.figure(figsize=(8, 5))
    plt.plot(faults, avg_latencies, marker='o', linestyle='-', color='teal')
    plt.xlabel("Fault Number")
    plt.ylabel("Average Latency (ms)")
    plt.title("Fault Number vs. Average Latency")
    plt.grid(True)
    plt.tight_layout()
    plt.savefig("faults_vs_latency.png")
    plt.show()

    # throughput
    plt.figure(figsize=(8, 5))
    plt.plot(faults, throughputs, marker='s', linestyle='-', color='orange')
    plt.xlabel("Fault Number")
    plt.ylabel("Throughput (byte/ms)")
    plt.title("Fault Number vs. Throughput")
    plt.grid(True)
    plt.tight_layout()
    plt.savefig("faults_vs_throughput.png")
    plt.show()


if __name__ == '__main__':
    run()
