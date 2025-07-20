import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns


if __name__ == '__main__':

    # 加载数据
    df = pd.read_csv("../files/metric_0.csv", parse_dates=["propose_time", "commit_time", "latency_ms"])

    # 确保按时间排序
    df = df.sort_values("propose_time")

    df["commit_sec"] = df["commit_time"].dt.floor("1s")
    df["latency_ms"] = pd.to_numeric(df["latency_ms"], errors="coerce")
    avg_by_time = df.groupby("commit_sec")["latency_ms"].mean()
    print(avg_by_time)
    avg_by_time.plot(title="平均延迟（ms） vs 时间")

    # # 1️⃣ 延迟变化图（每个区块）
    # plt.figure(figsize=(10, 4))
    # sns.lineplot(x="index", y="latency_ms", data=df, marker="o")
    # plt.title("📊 HotStuff Block Latency Over Time")
    # plt.xlabel("Block Index")
    # plt.ylabel("Latency (ms)")
    # plt.grid(True)
    # plt.tight_layout()
    # plt.show()
    # plt.savefig("latency_plot.png")

    # 2️⃣ 吞吐量（每秒提交多少区块）
    df["commit_sec"] = df["commit_time"].dt.floor("1s")
    tps = df.groupby("commit_sec").size().reset_index(name="tps")

    plt.figure(figsize=(10, 4))
    sns.lineplot(x="commit_sec", y="tps", data=tps)
    plt.title("🔥 Throughput (TPS) Over Time")
    plt.xlabel("Time")
    plt.ylabel("Transactions per Second (TPS)")
    plt.xticks(rotation=45)
    plt.grid(True)
    plt.tight_layout()
    plt.show()
    plt.savefig("throughput_plot.png")

    # # 3️⃣ View 演化图（哪个时间进入哪个 view）
    # plt.figure(figsize=(10, 4))
    # sns.lineplot(x="propose_time", y="view", data=df, marker="o")
    # plt.title("📈 View Number Over Time")
    # plt.xlabel("Propose Time")
    # plt.ylabel("View")
    # plt.xticks(rotation=45)
    # plt.grid(True)
    # plt.tight_layout()
    # plt.show()
    # plt.savefig("view_plot.png")