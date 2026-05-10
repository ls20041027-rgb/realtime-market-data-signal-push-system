"""DLL 接收端：注册表查 DLL 路径、加载、建隐藏窗口、Windows 消息回调分发。

拆分动机：主链路只关心"建好 hwnd 就进 PumpMessages"，具体怎么建窗口、怎么
绑 wparam 到各解析器是独立职责，放在 dll/receiver.py 让 main.py 更薄。
"""

import ctypes
import winreg as reg

import win32api
import win32gui

from dll.extended_api import init_dll_functions
from models.constant import My_Msg_StkData
from models.market_protocol import (
    RCV_FENBIDATA,
    RCV_FILEDATA,
    RCV_FINANCEDATA,
    RCV_MKTTBLDATA,
    RCV_REPORT,
    RCV_WORK_SENDMSG,
)
from message_parser.fenbi import parse_fenbi_message
from message_parser.filedata import parse_filedata_message
from message_parser.finance import parse_finance_message
from message_parser.mkttbl import parse_mkttbl_message
from message_parser.report import parse_report_message
from utils.envelope import emit_system_event

_WPARAM_NAME_MAP = {
    RCV_REPORT: "RCV_REPORT",
    RCV_FILEDATA: "RCV_FILEDATA",
    RCV_FENBIDATA: "RCV_FENBIDATA",
    RCV_MKTTBLDATA: "RCV_MKTTBLDATA",
    RCV_FINANCEDATA: "RCV_FINANCEDATA",
}

dll = None


def _wnd_proc(hwnd, msg, wparam, lparam):
    if msg != My_Msg_StkData:
        return win32gui.DefWindowProc(hwnd, msg, wparam, lparam)

    try:
        if wparam == RCV_REPORT:
            parse_report_message(lparam)
        elif wparam == RCV_FENBIDATA:
            parse_fenbi_message(lparam)
        elif wparam == RCV_MKTTBLDATA:
            parse_mkttbl_message(lparam)
        elif wparam == RCV_FINANCEDATA:
            parse_finance_message(lparam)
        elif wparam == RCV_FILEDATA:
            parse_filedata_message(lparam)
        else:
            emit_system_event(
                "WARNING", "unknown_wparam",
                {"wparam": int(wparam), "wparam_name": _WPARAM_NAME_MAP.get(wparam, "UNKNOWN")},
            )
            return 0
    except Exception as exc:
        emit_system_event(
            "ERROR", "dll_parse_exception",
            {"wparam": int(wparam), "wparam_name": _WPARAM_NAME_MAP.get(wparam, "UNKNOWN"), "error": str(exc)},
        )
    return 0


def _get_dll_path_from_registry():
    try:
        reg_key = reg.OpenKey(
            reg.HKEY_LOCAL_MACHINE,
            r"SOFTWARE\WOW6432Node\stockdrv",
            0, reg.KEY_READ,
        )
        dll_path, _ = reg.QueryValueEx(reg_key, "Driver")
        reg.CloseKey(reg_key)
        return dll_path
    except FileNotFoundError:
        print("[ERROR] stockdrv not found in registry")
        return None
    except Exception as exc:
        print(f"[ERROR] registry read failed: {exc}")
        return None


def _load_dll():
    dll_path = _get_dll_path_from_registry()
    if not dll_path:
        return None
    try:
        loaded_dll = ctypes.CDLL(dll_path)
        print(f"[INFO] DLL loaded: {dll_path}")
        return loaded_dll
    except OSError as exc:
        print(f"[ERROR] DLL load failed: {exc}")
        return None


def _create_hidden_window():
    wc = win32gui.WNDCLASS()
    wc.lpfnWndProc = _wnd_proc
    wc.hInstance = win32api.GetModuleHandle(None)
    wc.lpszClassName = "MyPythonWndClass"
    try:
        win32gui.RegisterClass(wc)
    except Exception:
        pass
    return win32gui.CreateWindow(
        wc.lpszClassName, "MyPythonWindow",
        0, 0, 0, 0, 0, 0, 0, wc.hInstance, None,
    )


def initialize_dll_receiver():
    """加载 DLL、建隐藏窗口、调用 Stock_Init 把回调绑到该窗口。

    成功返回 hwnd，失败返回 None（调用方据此决定是否继续启动）。
    """
    global dll
    dll = _load_dll()
    if dll is None:
        return None
    dll = init_dll_functions(dll)
    hwnd = _create_hidden_window()
    init_result = dll.Stock_Init(hwnd, My_Msg_StkData, RCV_WORK_SENDMSG)
    if init_result <= 0:
        print(f"[ERROR] Stock_Init failed, return={init_result}")
        return None
    print("[INFO] DLL receiver initialized, listening for realtime data")
    return hwnd
