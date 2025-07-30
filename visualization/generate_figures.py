#!/usr/bin/env python3
# -*- coding:utf-8 -*-

"""
@author:    yanghu
@time:      2025/7/30
"""

import pandas as pd
import matplotlib.pyplot as plt
import glob
import re


'''
@Scenario A: all nodes are well

figure1: latency vs total nodes count
    line1: pure basic hotstuff
    line2: basic hotstuff + cogsworth(all the time)
    line3: basic hotstuff + cogsworth(pause)

figure2: throughput vs total nodes count
    line1: pure basic hotstuff
    line2: basic hotstuff + cogsworth(all the time)
    line3: basic hotstuff + cogsworth(pause)

---------------------------------------------------------

@Scenario B: f nodes are fault

figure1: latency vs total nodes count
    line1: pure basic hotstuff
    line2: basic hotstuff + cogsworth(all the time)
    line3: basic hotstuff + cogsworth(pause)

figure2: throughput vs total nodes count
    line1: pure basic hotstuff
    line2: basic hotstuff + cogsworth(all the time)
    line3: basic hotstuff + cogsworth(pause)

---------------------------------------------------------

@Scenario C: fixed nodes count with unfixed fault count
exm: 10 nodes with {1|2|3} fault nodes
therefore in this figure, there should exists 5 lines

figure1: latency vs total nodes count
    line1: pure basic hotstuff
    line2: basic hotstuff + cogsworth(all the time)
    line3: basic hotstuff + cogsworth(pause)

figure2: throughput vs total nodes count
    line1: pure basic hotstuff
    line2: basic hotstuff + cogsworth(all the time)
    line3: basic hotstuff + cogsworth(pause)
'''


def filter_file(scenario):
    if scenario == "a":
        pattern = r"../files/metric_with_total_(\d+)_fault_0_(with|without)_pacemaker\.csv"

        csv_files = glob.glob("../files/metric_with_total_*_fault_0_*_pacemaker.csv")
        csv_files.sort()
        for file in csv_files:
            match = re.match(pattern, file)
            total_number = int(match.group(1))
            pacemaker = match.group(2)

            yield file, total_number, 0, pacemaker

    elif scenario == "b":
        pattern = r"../files/metric_with_total_(\d+)_fault_(\d+)_(with|without)_pacemaker\.csv"

        # 匹配所有数据文件
        csv_files = glob.glob("../files/metric_with_total_*_fault_*_*_pacemaker.csv")
        csv_files.sort()
        for file in csv_files:
            match = re.match(pattern, file)
            total_number = int(match.group(1))
            fault_number = int(match.group(2))
            pacemaker = match.group(3)

            if fault_number == 0:
                continue

            if total_number != 3 * fault_number + 1:
                continue

            yield file, total_number, fault_number, pacemaker

    elif scenario == "c":
        pattern = r"../files/metric_with_total_(\d+)_fault_(\d+)_(with|without)_pacemaker\.csv"

        # 匹配所有数据文件
        csv_files = glob.glob("../files/metric_with_total_*_fault_*_*_pacemaker.csv")
        csv_files.sort()
        for file in csv_files:
            match = re.match(pattern, file)
            total_number = int(match.group(1))
            fault_number = int(match.group(2))
            pacemaker = match.group(3)

            if fault_number == 0:
                continue

            yield file, total_number, fault_number, pacemaker

    return


def read_file(file):
    df = pd.read_csv(file)
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

    # 吞吐量计算（单位：byte/ms = kb/s）
    total_payload_byte = df["payload"].sum()
    total_latency_ms = df["latency_ms"].sum()
    throughput = total_payload_byte / total_latency_ms

    return avg_latency, throughput


def read_all_files(scenario: str):
    data = []

    for file, total_number, fault_number, pacemaker in filter_file(scenario):

        avg_latency, throughput = read_file(file)

        data.append({
            'total_number': total_number,
            'fault_number': fault_number,
            'pacemaker': pacemaker,
            'latency_ms': avg_latency,
            'throughput': throughput,
        })

    return data


def generate_plot(df_all: pd.DataFrame, x_label, scenario: str, key: str):

    plt.figure(figsize=(10, 6))
    for status in ['with', 'without']:
        df_subset = df_all[df_all['pacemaker'] == status]
        label = f"basic_hotstuff" if status != "with" else "bhs_with_pacemaker"
        plt.plot(df_subset[key], df_subset['latency_ms'], marker='o', label=label)
        plt.xticks(sorted(df_subset[key].unique()))  # 保证每个节点数都显示


    plt.title(f'Latency vs {x_label}')
    plt.xlabel(x_label)
    plt.ylabel(f"Average Latency (ms)")
    plt.legend()
    plt.grid(True)
    plt.tight_layout()
    plt.savefig(f"scenario_{scenario}_latency.png")
    plt.show()

    plt.figure(figsize=(10, 6))
    for status in ['with', 'without']:
        df_subset = df_all[df_all['pacemaker'] == status]
        label = f"basic_hotstuff" if status != "with" else "bhs_with_pacemaker"
        plt.plot(df_subset[key], df_subset['throughput'], marker='o', label=label)
        plt.xticks(sorted(df_subset[key].unique()))  # 保证每个节点数都显示

    plt.title(f'Throughput vs {x_label}')
    plt.xlabel(x_label)
    plt.ylabel(f"Throughput (kb/s)")
    plt.legend()
    plt.grid(True)
    plt.tight_layout()
    plt.savefig(f"scenario_{scenario}_throughput.png")
    plt.show()


def generate_plot_for_c(df_all: pd.DataFrame, x_label, scenario: str, key: str):

    plt.figure(figsize=(10, 6))
    for status in ['with', 'without']:
        for fault_number in range(1, 6):
            df_subset = df_all[(df_all['pacemaker'] == status) & (df_all["fault_number"] == fault_number)]
            label = f"basic_hotstuff_fault_{fault_number}" if status != "with" else f"bhs_fault_{fault_number}_with_pacemaker"
            plt.plot(df_subset[key], df_subset['latency_ms'], marker='o', label=label)
            plt.xticks(list(range(4, 17)))  # 保证每个节点数都显示


    plt.title(f'Latency vs {x_label}')
    plt.xlabel(x_label)
    plt.ylabel(f"Average Latency (ms)")
    plt.legend()
    plt.grid(True)
    plt.tight_layout()
    plt.savefig(f"scenario_{scenario}_latency.png")
    plt.show()

    plt.figure(figsize=(10, 6))
    for status in ['with', 'without']:
        for fault_number in range(1, 6):
            df_subset = df_all[(df_all['pacemaker'] == status) & (df_all["fault_number"] == fault_number)]
            label = f"basic_hotstuff_fault_{fault_number}" if status != "with" else f"bhs_fault_{fault_number}_with_pacemaker"
            plt.plot(df_subset[key], df_subset['throughput'], marker='o', label=label)
            plt.xticks(list(range(4, 17)))  # 保证每个节点数都显示

    plt.title(f'Throughput vs {x_label}')
    plt.xlabel(x_label)
    plt.ylabel(f"Throughput (kb/s)")
    plt.legend()
    plt.grid(True)
    plt.tight_layout()
    plt.savefig(f"scenario_{scenario}_throughput.png")
    plt.show()


def generate_figure(scenario: str = "a"):

    if scenario == "a":
        data = read_all_files(scenario=scenario)

        # 转换成 DataFrame 并排序
        df_all = pd.DataFrame(data)
        df_all.sort_values(by='total_number', inplace=True)

        generate_plot(df_all, "Total Nodes Count", scenario, "total_number")

    elif scenario == "b":
        data = read_all_files(scenario=scenario)

        # 转换成 DataFrame 并排序
        df_all = pd.DataFrame(data)
        df_all.sort_values(by='fault_number', inplace=True)

        generate_plot(df_all, "Fault Nodes Count", scenario, "fault_number")

    elif scenario == "c":
        data = read_all_files(scenario=scenario)

        # 转换成 DataFrame 并排序
        df_all = pd.DataFrame(data)
        df_all.sort_values(by='total_number', inplace=True)

        generate_plot_for_c(df_all, "Total Nodes Count", scenario, "total_number")




if __name__ == '__main__':
    # generate_figure("b")
    generate_figure("c")
