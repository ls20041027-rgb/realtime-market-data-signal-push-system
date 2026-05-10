"""上游 DLL 数据接入的**跨平台**协议常量与 ctypes 结构体定义。

为什么单独成文件：原 ``constant.py`` 里把 Windows-only 的 ``My_Msg_StkData``
（依赖 ``win32con``）和跨平台的市场字节常量、``RCV_*`` 消息类型、``ctypes``
结构体糊在一起，导致 Linux/macOS 的下游服务（stream_engine）只要 ``import``
就会炸在 ``win32con`` 上，进而被迫用 AST 解析源码来兜。本文件只放 **任何平台
都能安全 import** 的纯数据，Windows 专属 API 全部留在 ``constant.py``。
"""
import ctypes

ROOT_PATH = './data'

csv_RCV_REPORT = ["结构大小", "成交时间", "股票市场类型", "股票代码", "股票名称", "昨收", "今开", "最高", "最低", "最新", 
                "成交量", "成交额", "申买价1", "申买价2", "申买价3", "申买量1", "申买量2", "申买量3", 
                "申卖价1", "申卖价2", "申卖价3", "申卖量1", "申卖量2", "申卖量3", 
                "申买价4", "申买量4", "申卖价4", "申卖量4", 
                "申买价5", "申买量5", "申卖价5", "申卖量5"]
csv_RCV_FENBI = ["股票代码", "市场类型","日期",'昨收','今开','分别数量','具体时间','最高价' ,'最低价','最新价格',"成交量", "成交额", "申买价1", "申买价2", 
                "申买价3", "申买价4", "申买价5", "申买量1", "申买量2", "申买量3", 
                "申买量4", "申买量5", "申卖价1", "申卖价2", "申卖价3", "申卖价4", 
                "申卖价5", "申卖量1", "申卖量2", "申卖量3", "申卖量4", "申卖量5"]
csv_RCV_POWER_EX = ["股票代码", "市场类型","时间", "每股送", "每股配", "配股价",'每股红利',]
csv_RCV_MKTTBLDATA = ["市场代码", "市场名称", "市场属性", "数据日期", 
                    "交易时段个数", "开市时间", "收市时间", "证券个数"]
csv_TABLE_STRUCT = ['股票代码','股票名称',"每股手数"]
csv_FINANCEDATA = ["股票市场类型", "保留字段", "股票代码", "财务数据的日期", "总股本", "国家股", "发起人法人股",
                    "法人股", "B股", "H股", "目前流通", "职工股", "A2转配股", "总资产(千元)", "流动资产",
                    "固定资产", "无形资产", "长期投资", "流动负债", "长期负债", "资本公积金", "每股公积金",
                    "股东权益", "主营收入", "主营利润", "其他利润", "营业利润", "投资收益", "补贴收入",
                    "营业外收支", "上年损益调整", "利润总额", "税后利润", "净利润", "未分配利润", "每股未分配",
                    "每股收益", "每股净资产", "调整每股净资产", "股东权益比", "净资收益率"]
csv_MINUTES_EX = ["股票代码", "市场类型","时间", "最新价", "成交量", "成交金额"]
csv_DAY_EX = ["股票代码", "市场类型","时间", "开盘", "最高", "最低",'收盘','交易量','交易额','涨数','跌数']
csv_5MIN_EX = ["股票代码", "市场类型","时间", "开盘", "最高", "最低",'收盘','交易量','交易额','主动买量']

RCV_REPORT = 0x3f001234
RCV_FILEDATA = 0x3f001235
RCV_FENBIDATA = 0x3f001258
RCV_MKTTBLDATA = 0x3f001259
RCV_FINANCEDATA = 0x3f001300

RCV_WORK_SENDMSG = 4

SH_MARKET_EX = b'HS'
SZ_MARKET_EX = b'ZS'
HK_MARKET_EX = b'KH'

FILE_HISTORY_EX = 2
FILE_MINUTE_EX = 4
FILE_POWER_EX = 6
FILE_5MINUTE_EX = 81

FILE_BASE_EX = 0x1000
FILE_NEWS_EX = 0x1002
FILE_HTML_EX = 0x1004
FILE_TYPE_RES = -1

News_Sha_Ex = 2
News_Szn_Ex = 4
News_Fin_Ex = 6
News_TVSta_Ex = 8
News_Unknown_Ex = -1

RI_IDSTRING = 1
RI_IDCODE = 2
RI_VERSION = 3
RI_V2SUPPORT = 6

STKLABEL_LEN = 10
STKNAME_LEN = 32
MKTNAME_LEN = 16

EKE_HEAD_TAG = 0xffffffff


class RCV_FILE_HEADEx(ctypes.Structure):
    _fields_ = [
        ("m_dwAttrib", ctypes.c_uint32),
        ("m_dwLen", ctypes.c_uint32),
        ("m_dwSerialNo", ctypes.c_uint32),
        ("m_szFileName", ctypes.c_char * 260),
    ]
import time

class RCV_REPORT_STRUCTExV3(ctypes.Structure):
    _pack_ = 1
    _fields_ = [
        ("m_cbSize", ctypes.c_uint16),
        ("m_time", ctypes.c_long),
        ("m_wMarket", ctypes.c_uint16),
        ("m_szLabel", ctypes.c_char * STKLABEL_LEN),
        ("m_szName", ctypes.c_char * STKNAME_LEN),
        ("m_fLastClose", ctypes.c_float),
        ("m_fOpen", ctypes.c_float),
        ("m_fHigh", ctypes.c_float),
        ("m_fLow", ctypes.c_float),
        ("m_fNewPrice", ctypes.c_float),
        ("m_fVolume", ctypes.c_float),
        ("m_fAmount", ctypes.c_float),
        ("m_fBuyPrice", ctypes.c_float * 3),
        ("m_fBuyVolume", ctypes.c_float * 3),
        ("m_fSellPrice", ctypes.c_float * 3),
        ("m_fSellVolume", ctypes.c_float * 3),
        ("m_fBuyPrice4", ctypes.c_float),
        ("m_fBuyVolume4", ctypes.c_float),
        ("m_fSellPrice4", ctypes.c_float),
        ("m_fSellVolume4", ctypes.c_float),
        ("m_fBuyPrice5", ctypes.c_float),
        ("m_fBuyVolume5", ctypes.c_float),
        ("m_fSellPrice5", ctypes.c_float),
        ("m_fSellVolume5", ctypes.c_float),
    ]

class RCV_EKE_HEADEx(ctypes.Structure):
    _fields_ = [
        ("m_dwHeadTag", ctypes.c_ulong),
        ("m_wMarket", ctypes.c_ushort),
        ("m_szLabel", ctypes.c_char * STKLABEL_LEN),
    ]
class RCV_HISTORY_STRUCTEx(ctypes.Union):
    class _Fields(ctypes.Structure):
        _fields_ = [
            ("m_time", ctypes.c_long),
            ("m_fOpen", ctypes.c_float),
            ("m_fHigh", ctypes.c_float),
            ("m_fLow", ctypes.c_float),
            ("m_fClose", ctypes.c_float),
            ("m_fVolume", ctypes.c_float),
            ("m_fAmount", ctypes.c_float),
            ("m_wAdvance", ctypes.c_ushort),
            ("m_wDecline", ctypes.c_ushort),
        ]

    _fields_ = [
        ("data", _Fields),
        ("m_head", RCV_EKE_HEADEx),
    ]
    _anonymous_ = ("data",)
class RCV_MINUTE_STRUCTEx(ctypes.Union):
    class InnerStruct(ctypes.Structure):
            _fields_ = [
            ("m_time", ctypes.c_long),
            ("m_fPrice", ctypes.c_float),
            ("m_fVolume", ctypes.c_float),
            ("m_fAmount", ctypes.c_float)
        ]
    _fields_ = [
        ("m_head", RCV_EKE_HEADEx),
        ("m_inner_struct", InnerStruct)
    ]
    _anonymous_ = ("m_inner_struct",)
class RCV_POWER_STRUCTEx(ctypes.Union):
    class _Fields(ctypes.Structure):
        _fields_ = [
            ("m_time", ctypes.c_long),
            ("m_fGive", ctypes.c_float),
            ("m_fPei", ctypes.c_float),
            ("m_fPeiPrice", ctypes.c_float),
            ("m_fProfit", ctypes.c_float),
        ]
    _fields_ = [
        ("data", _Fields),
        ("m_head", RCV_EKE_HEADEx),
    ]
    _anonymous_ = ("data",)
class RCV_HISMINUTE_STRUCTEx(ctypes.Union):
    class _Fields(ctypes.Structure):
        _fields_ = [
            ("m_time", ctypes.c_long),
            ("m_fOpen", ctypes.c_float),
            ("m_fHigh", ctypes.c_float),
            ("m_fLow", ctypes.c_float),
            ("m_fClose", ctypes.c_float),
            ("m_fVolume", ctypes.c_float),
            ("m_fAmount", ctypes.c_float),
            ("m_fActiveBuyVol", ctypes.c_float),
        ]

    _fields_ = [
        ("data", _Fields),
        ("m_head", RCV_EKE_HEADEx),
    ]
    _anonymous_ = ("data",)
class RCV_DATA_UNION(ctypes.Union):
    _fields_ = [
        ("m_pReportV3", ctypes.POINTER(RCV_REPORT_STRUCTExV3)),
        ("m_pDay", ctypes.POINTER(RCV_HISTORY_STRUCTEx)),
        ("m_pMinute", ctypes.POINTER(RCV_MINUTE_STRUCTEx)),
        ("m_pPower", ctypes.POINTER(RCV_POWER_STRUCTEx)),
        ("m_p5Min", ctypes.POINTER(RCV_HISMINUTE_STRUCTEx)),
        ("m_pData", ctypes.c_void_p),
    ]

class RCV_DATA(ctypes.Structure):
    _fields_ = [
        ("m_wDataType", ctypes.c_int),
        ("m_nPacketNum", ctypes.c_int),
        ("m_File", RCV_FILE_HEADEx),
        ("m_bDISK", ctypes.c_bool),
        ("data_union", RCV_DATA_UNION),
    ]
class RCV_TABLE_STRUCT(ctypes.Structure):
    _pack_ = 1
    _fields_ = [
        ("m_szLabel", ctypes.c_char * STKLABEL_LEN),
        ("m_szName", ctypes.c_char * STKNAME_LEN),
        ("m_cProperty", ctypes.c_ushort),
    ]

class HLMarketType(ctypes.Structure):
    _pack_ = 1
    _fields_ = [
        ("m_wMarket", ctypes.c_ushort),
        ("m_Name", ctypes.c_char * MKTNAME_LEN),
        ("m_lProperty", ctypes.c_ulong),
        ("m_lDate", ctypes.c_ulong),
        ("m_PeriodCount", ctypes.c_ushort),
        ("m_OpenTime", ctypes.c_ushort * 5),
        ("m_CloseTime", ctypes.c_ushort * 5),
        ("m_nCount", ctypes.c_ushort),
        ("m_Data", ctypes.POINTER(RCV_TABLE_STRUCT)),
    ]

class Fin_LJF_STRUCTEx(ctypes.Structure):
    _pack_ = 1
    _fields_ = [
        ("m_wMarket", ctypes.c_ushort),
        ("N1", ctypes.c_ushort),
        ("m_szLabel", ctypes.c_char * 10),
        ("BGRQ", ctypes.c_long),
        ("ZGB", ctypes.c_float),
        ("GJG", ctypes.c_float),
        ("FQFRG", ctypes.c_float),
        ("FRG", ctypes.c_float),
        ("BGS", ctypes.c_float),
        ("HGS", ctypes.c_float),
        ("MQLT", ctypes.c_float),
        ("ZGG", ctypes.c_float),
        ("A2ZPG", ctypes.c_float),
        ("ZZC", ctypes.c_float),
        ("LDZC", ctypes.c_float),
        ("GDZC", ctypes.c_float),
        ("WXZC", ctypes.c_float),
        ("CQTZ", ctypes.c_float),
        ("LDFZ", ctypes.c_float),
        ("CQFZ", ctypes.c_float),
        ("ZBGJJ", ctypes.c_float),
        ("MGGJJ", ctypes.c_float),
        ("GDQY", ctypes.c_float),
        ("ZYSR", ctypes.c_float),
        ("ZYLR", ctypes.c_float),
        ("QTLR", ctypes.c_float),
        ("YYLR", ctypes.c_float),
        ("TZSY", ctypes.c_float),
        ("BTSR", ctypes.c_float),
        ("YYWSZ", ctypes.c_float),
        ("SNSYTZ", ctypes.c_float),
        ("LRZE", ctypes.c_float),
        ("SHLR", ctypes.c_float),
        ("JLR", ctypes.c_float),
        ("WFPLR", ctypes.c_float),
        ("MGWFP", ctypes.c_float),
        ("MGSY", ctypes.c_float),
        ("MGJZC", ctypes.c_float),
        ("TZMGJZC", ctypes.c_float),
        ("GDQYB", ctypes.c_float),
        ("JZCSYL", ctypes.c_float),
    ]

class RCV_FENBI_STRUCTEx(ctypes.Structure):
    _fields_ = [
        ("m_lTime", ctypes.c_long),
        ("m_fHigh", ctypes.c_float),
        ("m_fLow", ctypes.c_float),
        ("m_fNewPrice", ctypes.c_float),
        ("m_fVolume", ctypes.c_float),
        ("m_fAmount", ctypes.c_float),
        ("m_lStroke", ctypes.c_long),
        ("m_fBuyPrice", ctypes.c_float * 5),
        ("m_fBuyVolume", ctypes.c_float * 5),
        ("m_fSellPrice", ctypes.c_float * 5),
        ("m_fSellVolume", ctypes.c_float * 5),
    ]

class RCV_FENBI(ctypes.Structure):
    _fields_ = [
        ("m_wMarket", ctypes.c_ushort),
        ("m_szLabel", ctypes.c_char * STKLABEL_LEN),
        ("m_lDate", ctypes.c_long),
        ("m_fLastClose", ctypes.c_float),
        ("m_fOpen", ctypes.c_float),
        ("m_nCount", ctypes.c_ushort),
        ("m_Data", ctypes.POINTER(RCV_FENBI_STRUCTEx)),
    ]

class DAT_FENBI(ctypes.Structure):
    _pack_ = 1
    _fields_ = [
        ("m_szLabel", ctypes.c_char * 10),
        ("m_wMarket", ctypes.c_ushort),
        ("m_lDate", ctypes.c_long),
        ("m_fLastClose", ctypes.c_float),
        ("m_fOpen", ctypes.c_float),
        ("m_nCount", ctypes.c_ushort),
        ("formatted_date", ctypes.c_char * 20),
        ("m_fHigh", ctypes.c_float),
        ("m_fLow", ctypes.c_float),
        ("m_fNewPrice", ctypes.c_float),
        ("m_fVolume", ctypes.c_float),
        ("m_fAmount", ctypes.c_float),

        ("m_fBuyPrice1", ctypes.c_float),
        ("m_fBuyPrice2", ctypes.c_float),
        ("m_fBuyPrice3", ctypes.c_float),
        ("m_fBuyPrice4", ctypes.c_float),
        ("m_fBuyPrice5", ctypes.c_float),

        ("m_fBuyVolume1", ctypes.c_float),
        ("m_fBuyVolume2", ctypes.c_float),
        ("m_fBuyVolume3", ctypes.c_float),
        ("m_fBuyVolume4", ctypes.c_float),
        ("m_fBuyVolume5", ctypes.c_float),

        ("m_fSellPrice1", ctypes.c_float),
        ("m_fSellPrice2", ctypes.c_float),
        ("m_fSellPrice3", ctypes.c_float),
        ("m_fSellPrice4", ctypes.c_float),
        ("m_fSellPrice5", ctypes.c_float),

        ("m_fSellVolume1", ctypes.c_float),
        ("m_fSellVolume2", ctypes.c_float),
        ("m_fSellVolume3", ctypes.c_float),
        ("m_fSellVolume4", ctypes.c_float),
        ("m_fSellVolume5", ctypes.c_float),
    ]

class DAT_REPORT(ctypes.Structure):
    _pack_ = 1
    _fields_ = [
    ("m_cbSize", ctypes.c_uint16),
    ("formatted_date", ctypes.c_char * 20),
    ("m_wMarket", ctypes.c_uint16),
    ("m_szLabel", ctypes.c_char * STKLABEL_LEN),
    ("m_szName", ctypes.c_char * STKNAME_LEN),
    ("m_fLastClose", ctypes.c_float),
    ("m_fOpen", ctypes.c_float),
    ("m_fHigh", ctypes.c_float),
    ("m_fLow", ctypes.c_float),
    ("m_fNewPrice", ctypes.c_float),
    ("m_fVolume", ctypes.c_float),
    ("m_fAmount", ctypes.c_float),
    ("m_fBuyPrice1", ctypes.c_float),
    ("m_fBuyPrice2", ctypes.c_float),
    ("m_fBuyPrice3", ctypes.c_float),
    ("m_fBuyVolume1", ctypes.c_float),
    ("m_fBuyVolume2", ctypes.c_float),
    ("m_fBuyVolume3", ctypes.c_float),
    ("m_fSellPrice1", ctypes.c_float),
    ("m_fSellPrice2", ctypes.c_float),
    ("m_fSellPrice3", ctypes.c_float),
    ("m_fSellVolume1", ctypes.c_float),
    ("m_fSellVolume2", ctypes.c_float),
    ("m_fSellVolume3", ctypes.c_float),
    ("m_fBuyPrice4", ctypes.c_float),
    ("m_fBuyVolume4", ctypes.c_float),
    ("m_fSellPrice4", ctypes.c_float),
    ("m_fSellVolume4", ctypes.c_float),
    ("m_fBuyPrice5", ctypes.c_float),
    ("m_fBuyVolume5", ctypes.c_float),
    ("m_fSellPrice5", ctypes.c_float),
    ("m_fSellVolume5", ctypes.c_float),
]

class DAT_POWER_EX(ctypes.Structure):
    _pack_ = 1
    _fields_ = [
    ("m_szLabel", ctypes.c_char * STKLABEL_LEN),
    ("m_wMarket", ctypes.c_ushort),
    ("formatted_date", ctypes.c_char * 20),
    ("m_fGive", ctypes.c_float),
    ("m_fPei", ctypes.c_float),
    ("m_fPeiPrice", ctypes.c_float),
    ("m_fProfit", ctypes.c_float),
    ]

class DAT_HLMarketType(ctypes.Structure):
    _pack_ = 1
    _fields_ = [
        ("m_wMarket", ctypes.c_ushort),
        ("m_Name", ctypes.c_char * MKTNAME_LEN),
        ("m_lProperty", ctypes.c_ulong),
        ("m_lDate", ctypes.c_ulong),
        ("m_PeriodCount", ctypes.c_ushort),
        ("m_OpenTime", ctypes.c_ushort * 5),
        ("m_CloseTime", ctypes.c_ushort * 5),
        ("m_nCount", ctypes.c_ushort),
    ]

class DAT_TABLE_STRUCT(ctypes.Structure):
    _pack_ = 1
    _fields_ = [
        ("m_szLabel", ctypes.c_char * STKLABEL_LEN),
        ("m_szName", ctypes.c_char * STKNAME_LEN),
        ("m_cProperty", ctypes.c_ushort),
    ]
class DAT_FINANCEDATA(ctypes.Structure):
    _pack_ = 1
    _fields_ = [
        ("m_wMarket", ctypes.c_ushort),
        ("N1", ctypes.c_ushort),
        ("m_szLabel", ctypes.c_char * 10),
        ("formatted_date", ctypes.c_char * 20),
        ("ZGB", ctypes.c_float),
        ("GJG", ctypes.c_float),
        ("FQFRG", ctypes.c_float),
        ("FRG", ctypes.c_float),
        ("BGS", ctypes.c_float),
        ("HGS", ctypes.c_float),
        ("MQLT", ctypes.c_float),
        ("ZGG", ctypes.c_float),
        ("A2ZPG", ctypes.c_float),
        ("ZZC", ctypes.c_float),
        ("LDZC", ctypes.c_float),
        ("GDZC", ctypes.c_float),
        ("WXZC", ctypes.c_float),
        ("CQTZ", ctypes.c_float),
        ("LDFZ", ctypes.c_float),
        ("CQFZ", ctypes.c_float),
        ("ZBGJJ", ctypes.c_float),
        ("MGGJJ", ctypes.c_float),
        ("GDQY", ctypes.c_float),
        ("ZYSR", ctypes.c_float),
        ("ZYLR", ctypes.c_float),
        ("QTLR", ctypes.c_float),
        ("YYLR", ctypes.c_float),
        ("TZSY", ctypes.c_float),
        ("BTSR", ctypes.c_float),
        ("YYWSZ", ctypes.c_float),
        ("SNSYTZ", ctypes.c_float),
        ("LRZE", ctypes.c_float),
        ("SHLR", ctypes.c_float),
        ("JLR", ctypes.c_float),
        ("WFPLR", ctypes.c_float),
        ("MGWFP", ctypes.c_float),
        ("MGSY", ctypes.c_float),
        ("MGJZC", ctypes.c_float),
        ("TZMGJZC", ctypes.c_float),
        ("GDQYB", ctypes.c_float),
        ("JZCSYL", ctypes.c_float),
    ]

class DAT_MINUTE_EX(ctypes.Structure):
    _pack_ = 1
    _fields_ = [
        ("m_szLabel", ctypes.c_char * 10),
        ("m_wMarket", ctypes.c_ushort),
        ("formatted_date", ctypes.c_char * 20),
        ("m_fPrice", ctypes.c_float),
        ("m_fVolume", ctypes.c_float),
        ("m_fAmount", ctypes.c_float),
    ]

class DAT_5MINUTE_EX(ctypes.Structure):
    _pack_ = 1
    _fields_ = [
        ("m_szLabel", ctypes.c_char * 10),
        ("m_wMarket", ctypes.c_ushort),
        ("formatted_date", ctypes.c_char * 20),
        ("m_fOpen", ctypes.c_float),
        ("m_fHigh", ctypes.c_float),
        ("m_fLow", ctypes.c_float),
        ("m_fClose", ctypes.c_float),
        ("m_fVolume", ctypes.c_float),
        ("m_fAmount", ctypes.c_float),
        ("m_fActiveBuyVol", ctypes.c_float),
    ]
class DAT_DAY_EX(ctypes.Structure):
    _pack_ = 1
    _fields_ = [
        ("m_szLabel", ctypes.c_char * 10),
        ("m_wMarket", ctypes.c_ushort),
        ("formatted_date", ctypes.c_char * 20),
        ("m_fOpen", ctypes.c_float),
        ("m_fHigh", ctypes.c_float),
        ("m_fLow", ctypes.c_float),
        ("m_fClose", ctypes.c_float),
        ("m_fVolume", ctypes.c_float),
        ("m_fAmount", ctypes.c_float),
        ("m_wAdvance", ctypes.c_ushort),
        ("m_wDecline", ctypes.c_ushort),
        ]
