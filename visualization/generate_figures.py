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
    todo: line3: basic hotstuff + cogsworth(pause)

figure2: throughput vs total nodes count
    line1: pure basic hotstuff
    line2: basic hotstuff + cogsworth(all the time)
    todo: line3: basic hotstuff + cogsworth(pause)

---------------------------------------------------------

@Scenario B: f nodes are fault

figure1: latency vs total nodes count
    line1: pure basic hotstuff
    line2: basic hotstuff + cogsworth(all the time)
    todo: line3: basic hotstuff + cogsworth(pause)

figure2: throughput vs total nodes count
    line1: pure basic hotstuff
    line2: basic hotstuff + cogsworth(all the time)
    todo: line3: basic hotstuff + cogsworth(pause)

---------------------------------------------------------

@Scenario C: fixed nodes count with unfixed fault count
exm: 10 nodes with {1|2|3} fault nodes
therefore in this figure, there should exists 5 lines

figure1: latency vs total nodes count
    line1: pure basic hotstuff
    line2: basic hotstuff + cogsworth(all the time)
    todo: line3: basic hotstuff + cogsworth(pause)

figure2: throughput vs total nodes count
    line1: pure basic hotstuff
    line2: basic hotstuff + cogsworth(all the time)
    todo: line3: basic hotstuff + cogsworth(pause)
'''


class VisualizationBase:
    scenario: str

    def generate_figure(self):
        raise NotImplementedError

    def filter_file(self):
        raise NotImplementedError

    @staticmethod
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


    def read_all_files(self):
        data = []

        for file, total_number, fault_number, pacemaker in self.filter_file():

            avg_latency, throughput = self.read_file(file)

            data.append({
                'total_number': total_number,
                'fault_number': fault_number,
                'pacemaker': pacemaker,
                'latency_ms': avg_latency,
                'throughput': throughput,
            })

        return data

    @classmethod
    def generate_plot(cls, df_all: pd.DataFrame, x_label, key: str):

        plt.figure(figsize=(10, 6))
        for status in ['with', 'without']:
            df_subset = df_all[df_all['pacemaker'] == status]
            label = f"basic_hotstuff" if status != "with" else "basic_hotstuff_with_cogsworth"
            plt.plot(df_subset[key], df_subset['latency_ms'], marker='o', label=label)
            plt.xticks(sorted(df_subset[key].unique()))  # 保证每个节点数都显示


        plt.title(f'Latency vs {x_label}')
        plt.xlabel(x_label)
        plt.ylabel(f"Average Latency (ms)")
        plt.legend()
        plt.grid(True)
        plt.tight_layout()
        plt.savefig(f"scenario_{cls.scenario}_latency.png")
        plt.show()

        plt.figure(figsize=(10, 6))
        for status in ['with', 'without']:
            df_subset = df_all[df_all['pacemaker'] == status]
            label = f"basic_hotstuff" if status != "with" else "basic_hotstuff_with_cogsworth"
            plt.plot(df_subset[key], df_subset['throughput'], marker='o', label=label)
            plt.xticks(sorted(df_subset[key].unique()))  # 保证每个节点数都显示

        plt.title(f'Throughput vs {x_label}')
        plt.xlabel(x_label)
        plt.ylabel(f"Throughput (kb/s)")
        plt.legend()
        plt.grid(True)
        plt.tight_layout()
        plt.savefig(f"scenario_{cls.scenario}_throughput.png")
        plt.show()

    def calculate_latency_increase(self, df_all):
        print(df_all.columns)
        group_keys = ['total_number', 'fault_number']
        pivot_latency = df_all.pivot_table(index=group_keys, columns='pacemaker', values='latency_ms')
        pivot_latency = pivot_latency.dropna(subset=['with', 'without'])
        pivot_latency['latency_increase_percent'] = (pivot_latency['with'] - pivot_latency['without']) / pivot_latency['without'] * 100
        mean_increase = pivot_latency['latency_increase_percent'].mean()
        print(f"Average latency increase with Cogsworth: {mean_increase:.2f}%")

        # throughput
        pivot_throughput = df_all.pivot_table(index=group_keys, columns='pacemaker', values='throughput')
        pivot_throughput = pivot_throughput.dropna(subset=['with', 'without'])
        # throughput下降用 (with - without)/without * 100，正为提升，负为下降
        pivot_throughput['throughput_decrease_percent'] = (pivot_throughput['with'] - pivot_throughput['without']) / pivot_throughput['without'] * 100
        mean_decrease = pivot_throughput['throughput_decrease_percent'].mean()
        print(f"Throughput increase with Cogsworth: {mean_decrease:.2f}%")

        return mean_increase, mean_decrease



class VisualiseA(VisualizationBase):
    scenario = "a"

    def filter_file(self):
        pattern = r"../files/metric_with_total_(\d+)_fault_0_(with|without)_cogsworth\.csv"

        csv_files = glob.glob("../files/metric_with_total_*_fault_0_*_cogsworth.csv")
        csv_files.sort()
        for file in csv_files:
            match = re.match(pattern, file)
            total_number = int(match.group(1))
            pacemaker = match.group(2)

            yield file, total_number, 0, pacemaker

    def generate_figure(self):
        data = self.read_all_files()

        # 转换成 DataFrame 并排序
        df_all = pd.DataFrame(data)
        df_all.sort_values(by='total_number', inplace=True)

        self.calculate_latency_increase(df_all)

        self.generate_plot(df_all, "Total Nodes Count", "total_number")


class VisualiseB(VisualizationBase):
    scenario = "b"

    def filter_file(self):
        pattern = r"../files/metric_with_total_(\d+)_fault_(\d+)_(with|without)_cogsworth\.csv"

        # 匹配所有数据文件
        csv_files = glob.glob("../files/metric_with_total_*_fault_*_*_cogsworth.csv")
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

    def generate_figure(self):
        data = self.read_all_files()

        # 转换成 DataFrame 并排序
        df_all = pd.DataFrame(data)
        df_all.sort_values(by='fault_number', inplace=True)

        self.calculate_latency_increase(df_all)

        self.generate_plot(df_all, "Fault Nodes Count", "fault_number")


class VisualiseC(VisualizationBase):
    scenario = "c"

    def filter_file(self):
        pattern = r"../files/metric_with_total_(\d+)_fault_(\d+)_(with|without)_cogsworth\.csv"

        # 匹配所有数据文件
        csv_files = glob.glob("../files/metric_with_total_*_fault_*_*_cogsworth.csv")
        csv_files.sort()
        for file in csv_files:
            match = re.match(pattern, file)
            total_number = int(match.group(1))
            fault_number = int(match.group(2))
            pacemaker = match.group(3)

            if fault_number == 0:
                continue

            if fault_number == 5:
                continue

            if total_number < 10:
                continue

            yield file, total_number, fault_number, pacemaker

    def generate_figure(self):
        data = self.read_all_files()

        # 转换成 DataFrame 并排序
        df_all = pd.DataFrame(data)
        df_all.sort_values(by='total_number', inplace=True)

        self.calculate_latency_increase(df_all)

        self.generate_plot(df_all, "Total Nodes Count", "total_number")

    @classmethod
    def generate_plot(cls, df_all: pd.DataFrame, x_label, key: str):

        plt.figure(figsize=(10, 6))
        for status in ['with', 'without']:
            for fault_number in range(1, 5):
                df_subset = df_all[(df_all['pacemaker'] == status) & (df_all["fault_number"] == fault_number)]
                label = f"basic_hotstuff_fault_{fault_number}" if status != "with" else f"basic_hotstuff_fault_{fault_number}_with_cogsworth"
                plt.plot(df_subset[key], df_subset['latency_ms'], marker='o', label=label)
                plt.xticks(list(range(7, 16)))  # 保证每个节点数都显示


        plt.title(f'Latency vs {x_label}')
        plt.xlabel(x_label)
        plt.ylabel(f"Average Latency (ms)")
        plt.legend()
        plt.grid(True)
        plt.tight_layout()
        plt.savefig(f"scenario_{cls.scenario}_latency.png")
        plt.show()

        plt.figure(figsize=(10, 6))
        for status in ['with', 'without']:
            for fault_number in range(1, 5):
                df_subset = df_all[(df_all['pacemaker'] == status) & (df_all["fault_number"] == fault_number)]
                label = f"basic_hotstuff_fault_{fault_number}" if status != "with" else f"basic_hotstuff_fault_{fault_number}_with_cogsworth"
                plt.plot(df_subset[key], df_subset['throughput'], marker='o', label=label)
                plt.xticks(list(range(7, 16)))  # 保证每个节点数都显示

        plt.title(f'Throughput vs {x_label}')
        plt.xlabel(x_label)
        plt.ylabel(f"Throughput (kb/s)")
        plt.legend()
        plt.grid(True)
        plt.tight_layout()
        plt.savefig(f"scenario_{cls.scenario}_throughput.png")
        plt.show()


if __name__ == '__main__':
    # VisualiseA().generate_figure()
    # VisualiseB().generate_figure()
    VisualiseC().generate_figure()
