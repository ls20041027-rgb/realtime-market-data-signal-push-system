"""上游 DLL 数据接入常量（Windows 专属入口）。

本文件只剩 Windows-only 的一个自定义消息号；跨平台的协议常量 / ctypes 结构体
全部在同目录的 :mod:`market_protocol`。跨服务/跨平台引用请直接 import
``market_protocol``；**本文件不再对外 re-export 任何跨平台符号**（R19 禁 import *
之后，旧的"转出别名"模式没有保留价值，显式 import 更清晰）。
"""
import win32con

My_Msg_StkData = win32con.WM_APP + 1
