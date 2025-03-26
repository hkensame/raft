#!/bin/bash

# 获取与 main.go 相关的所有进程 ID
pids=$(ps -lfe | grep "main.go" | grep -v "grep" | awk '{print $4}')

# 检查是否有进程需要被杀死
if [ -z "$pids" ]; then
    echo "没有找到匹配的进程"
    exit 0
fi

# 杀死所有找到的进程
for pid in $pids; do
    echo "正在杀死进程 PID: $pid"
    kill -9 $pid
done

echo "所有相关进程已被杀死"
