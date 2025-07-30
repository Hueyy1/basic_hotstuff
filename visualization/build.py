#!/usr/bin/env python3
# -*- coding:utf-8 -*-

"""
@author:    yanghu
@time:      2025/7/30
"""

import subprocess
import sys

SRC_MAIN = "../src/main"


def build():
    # 编译 Go 程序
    print("🔨 Building Go program...")
    build_cmd = ["go", "build", "-o", SRC_MAIN, "../src/main.go"]
    result = subprocess.run(build_cmd)
    if result.returncode != 0:
        print("❌ Go build failed.")
        sys.exit(1)
    print("✅ Build complete.")
