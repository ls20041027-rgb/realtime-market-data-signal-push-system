import ctypes


def decode_bytes(raw_value):
    """
    将 DLL 返回的字节串尽量解码为 Python 字符串。

    解码策略：
    1. 优先尝试 GBK，因为老式行情接口常见 GBK 编码；
    2. 再尝试 UTF-8，兼容后续新数据源；
    3. 最后回退到 latin1，避免因为编码异常直接丢失数据。
    """
    for encoding in ("gbk", "utf-8", "latin1"):
        try:
            return raw_value.decode(encoding).strip("\x00").strip()
        except Exception:
            continue

    return raw_value.decode("latin1", errors="ignore").strip("\x00").strip()


def struct_to_dict(ctypes_struct):
    """
    将 ctypes 结构体递归转换为 Python 字典。

    这是数据接入层的通用转换函数，主要负责：
    - 处理 `bytes` 到字符串的自动解码；
    - 处理嵌套 `Structure / Union`；
    - 处理 `Array` 字段；
    - 将指针转换为可序列化的地址值。
    - 处理 `_anonymous_` 属性，将匿名字段展平到顶层。
    """
    result = {}

    anonymous = set(getattr(ctypes_struct, "_anonymous_", ()) or ())

    for field_name, field_type in ctypes_struct._fields_:
        value = getattr(ctypes_struct, field_name)

        if isinstance(value, bytes):
            result[field_name] = decode_bytes(value)
            continue

        if isinstance(value, (ctypes.Structure, ctypes.Union)):
            sub = struct_to_dict(value)
            if field_name in anonymous:
                result.update(sub)
            else:
                result[field_name] = sub
            continue

        if isinstance(value, ctypes.Array):
            if issubclass(field_type._type_, (ctypes.Structure, ctypes.Union)):
                result[field_name] = [struct_to_dict(item) for item in value]
            else:
                result[field_name] = [item for item in value]
            continue

        if isinstance(value, ctypes._Pointer):
            result[field_name] = ctypes.cast(value, ctypes.c_void_p).value
            continue

        result[field_name] = value

    return result
