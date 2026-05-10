"""data_ingestion 服务入口（简化版）。

职责：只负责装配主链路，不承载业务细节：
  1. 校验 Windows 平台；
  2. 拉起 Kafka 生产者线程（消费共享队列）；
  3. 初始化 DLL 接收窗口并进入 Windows 消息循环；
  4. 捕获 Ctrl-C / 退出信号后置位 stop_event，优雅收尾。

业务细节分散在：
  - dll/        —— DLL 签名、加载、窗口、wnd_proc 回调
  - parser/     —— 5 类 DLL 消息的结构体解析
  - producer/   —— Kafka 生产者线程 + 队列 + 停机信号
  - utils/      —— 消息封装、符号抽取、系统事件、ctypes 转 dict
"""

import platform
import sys
import threading

from dll.receiver import initialize_dll_receiver
from producer.kafka_producer import kafka_producer_worker, stop_event


def main() -> int:
    if platform.system() != "Windows":
        print("[FATAL] data_ingestion requires Windows")
        return 1

    import win32gui

    producer_thread = threading.Thread(target=kafka_producer_worker, daemon=True)
    producer_thread.start()

    hwnd = initialize_dll_receiver()
    if hwnd is None:
        stop_event.set()
        print("[FATAL] data_ingestion startup failed")
        return 1

    print("[INFO] data_ingestion service started")
    try:
        win32gui.PumpMessages()
    except KeyboardInterrupt:
        print("[INFO] shutting down data_ingestion")
    finally:
        stop_event.set()
    return 0


if __name__ == "__main__":
    sys.exit(main())
