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

if __name__ == '__main__':

    # 匹配所有数据文件
    csv_files = glob.glob("metric_*.csv")

    # 结果列表
    results = []

    # 遍历每个文件
    for file in csv_files:
        try:
            # 提取节点数量（比如 latency_4nodes.csv → 4）
            match = re.search(r"(\d+)", file)
            if not match:
                continue
            node_count = int(match.group(1))

            df = pd.read_csv(file, parse_dates=["propose_time", "commit_time"])
            df["latency_ms"] = pd.to_numeric(df["latency_ms"], errors="coerce")
            df = df.dropna(subset=["latency_ms"])

            # === 计算平均延迟 ===
            avg_latency = df["latency_ms"].mean()

            # === 计算吞吐量（总交易数 / 时间跨度）===
            if len(df) > 1:
                duration = (df["commit_time"].max() - df["commit_time"].min()).total_seconds()
                tput = len(df) / duration if duration > 0 else 0
            else:
                tput = 0

            results.append({
                "nodes": node_count,
                "avg_latency_ms": avg_latency,
                "throughput_tps": tput
            })

        except Exception as e:
            print(f"❌ Error processing {file}: {str(e)}")

    # 构建 DataFrame 并排序
    result_df = pd.DataFrame(results).sort_values("nodes")

    # ✅ 打印结果
    print(result_df)

    # === 画图 ===
    sns.set(style="whitegrid")

    # 1. 平均延迟图
    plt.figure(figsize=(8, 4))
    sns.lineplot(x="nodes", y="avg_latency_ms", data=result_df, marker="o")
    plt.title("replicas vs latency")
    plt.xlabel("replicas")
    plt.ylabel("latency (ms)")
    plt.tight_layout()
    plt.savefig("latency_vs_replicas.png")
    plt.show()

    # 2. 吞吐量图
    plt.figure(figsize=(8, 4))
    sns.lineplot(x="nodes", y="throughput_tps", data=result_df, marker="o")
    plt.title("replicas vs throughput")
    plt.xlabel("replicas")
    plt.ylabel("throughput (tx/sec)")
    plt.tight_layout()
    plt.savefig("throughput_vs_replicas.png")
    plt.show()
