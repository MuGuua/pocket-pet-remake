#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
按旧 .dat 算法批量编码 / 解码当前目录文件（双击可用版）

用法：
- 直接双击运行
- 或命令行运行：python batch_dat_codec.py
- 按提示输入：
    1 = 批量编码当前目录普通文件为 .dat
    2 = 批量解码当前目录 .dat 文件

说明：
- 只处理当前脚本所在目录
- 会跳过脚本自己
- 默认删除原文件，只保留转换后的文件
"""

from pathlib import Path

SCRIPT_PATH = Path(__file__).resolve()
CURRENT_DIR = SCRIPT_PATH.parent


def decode_bytes(data: bytes) -> bytes:
    out = bytearray(data)
    key = len(out) & 0xFF
    for i in range(len(out) - 1, 0, -1):
        out[i] ^= out[i - 1]
    if out:
        out[0] ^= key
    return bytes(out)



def encode_bytes(data: bytes) -> bytes:
    if not data:
        return b""
    out = bytearray(len(data))
    key = len(data) & 0xFF
    out[0] = data[0] ^ key
    for i in range(1, len(data)):
        out[i] = data[i] ^ out[i - 1]
    return bytes(out)



def detect_kind(data: bytes) -> str:
    if len(data) >= 8 and data[:8] == b"\x89PNG\r\n\x1a\n":
        return ".png"
    if len(data) >= 4 and data[:4] == b"GIF8":
        return ".gif"
    if len(data) >= 3 and data[:3] == b"\xFF\xD8\xFF":
        return ".jpg"

    sample = data[:256]
    if not sample:
        return ".bin"

    printable = 0
    for b in sample:
        if 9 <= b <= 13 or 32 <= b <= 126 or b >= 128:
            printable += 1
    if printable / len(sample) >= 0.85:
        return ".txt"
    return ".bin"



def encode_file(file_path: Path):
    if file_path.resolve() == SCRIPT_PATH:
        return False
    if not file_path.is_file():
        return False
    if file_path.suffix.lower() == ".dat":
        return False

    out_path = file_path.with_suffix(".dat")
    raw = file_path.read_bytes()
    encoded = encode_bytes(raw)
    out_path.write_bytes(encoded)
    file_path.unlink()
    print(f"[编码] {file_path.name} -> {out_path.name}")
    return True



def decode_file(file_path: Path):
    if file_path.resolve() == SCRIPT_PATH:
        return False
    if not file_path.is_file():
        return False
    if file_path.suffix.lower() != ".dat":
        return False

    raw = file_path.read_bytes()
    decoded = decode_bytes(raw)
    new_ext = detect_kind(decoded)
    out_path = file_path.with_suffix(new_ext)
    out_path.write_bytes(decoded)
    file_path.unlink()
    print(f"[解码] {file_path.name} -> {out_path.name}")
    return True



def run_encode():
    count = 0
    for file_path in CURRENT_DIR.iterdir():
        if encode_file(file_path):
            count += 1
    print(f"\n批量编码完成，共处理 {count} 个文件。")



def run_decode():
    count = 0
    for file_path in CURRENT_DIR.iterdir():
        if decode_file(file_path):
            count += 1
    print(f"\n批量解码完成，共处理 {count} 个文件。")



def main():
    print("=" * 40)
    print(" KD .dat 批量编码/解码工具 ")
    print("=" * 40)
    print(f"当前目录: {CURRENT_DIR}")
    print()
    print("1 = 批量编码（普通文件 -> .dat）")
    print("2 = 批量解码（.dat -> 原文件）")
    print()

    choice = input("请输入 1 或 2，然后回车: ").strip()
    print()

    if choice == "1":
        run_encode()
    elif choice == "2":
        run_decode()
    else:
        print("输入错误，只能输入 1 或 2")

    print()
    input("按回车键退出...")


if __name__ == "__main__":
    main()
