"""stockdrv DLL 函数签名绑定。

本次改动动机：原文件除 `init_dll_functions` 外，还有 10 个形如
`ask_stock_day(dll, code, period): return dll.AskStockDay(code.encode('ascii'), period)`
的薄封装函数以及 `_set_signature` / `_encode_stock_code` 两个一行辅助函数。
全仓 grep 显示这些函数从未被任何地方调用，属于 R3/奥卡姆剃刀明确禁止的
"未被要求的抽象"。一次性删除，只保留签名注册逻辑本身。
"""

import ctypes
from ctypes import wintypes

BOOL = wintypes.BOOL


def init_dll_functions(dll):
    """初始化 stockdrv DLL 中本服务会用到的函数签名。"""
    dll.Stock_Init.argtypes = [ctypes.c_void_p, ctypes.c_uint, ctypes.c_int]
    dll.Stock_Init.restype = ctypes.c_int

    dll.SetupReceiver.argtypes = [ctypes.c_int]
    dll.SetupReceiver.restype = ctypes.c_int

    dll.AskStockDay.argtypes = [ctypes.c_char_p, ctypes.c_int]
    dll.AskStockDay.restype = ctypes.c_int

    dll.AskStockMn5.argtypes = [ctypes.c_char_p, ctypes.c_int]
    dll.AskStockMn5.restype = ctypes.c_int

    dll.AskStockBase.argtypes = [ctypes.c_char_p]
    dll.AskStockBase.restype = ctypes.c_int

    dll.AskStockNews.argtypes = []
    dll.AskStockNews.restype = ctypes.c_int

    dll.AskStockHalt.argtypes = []
    dll.AskStockHalt.restype = ctypes.c_int

    dll.AskStockMin.argtypes = [ctypes.c_char_p]
    dll.AskStockMin.restype = ctypes.c_int

    dll.AskStockPRP.argtypes = [ctypes.c_char_p]
    dll.AskStockPRP.restype = ctypes.c_int

    dll.AskStockPwr.argtypes = []
    dll.AskStockPwr.restype = ctypes.c_int

    dll.AskStockFin.argtypes = []
    dll.AskStockFin.restype = ctypes.c_int

    return dll
