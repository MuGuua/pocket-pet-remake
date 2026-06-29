class_name ServerErrorMessages
extends RefCounted

## 将服务端 ERROR_PUSH 的英文 msg 转为玩家可读的简体中文；未知文案原样返回。
static func format(error_code: int, message: String) -> String:
    var normalized: String = message.strip_edges().to_lower()
    match normalized:
        "player level too low":
            return "玩家等级不足，无法佩戴该装备。"
        "equipment is damaged":
            return "装备已损坏，请先修复后再佩戴。"
        "invalid equipment item":
            return "该物品无法作为装备佩戴。"
        "equipment slot mismatch":
            return "该装备与目标槽位不匹配。"
        "bag item not found":
            return "背包中找不到该物品，请刷新后重试。"
        "equipment slot empty":
            return "该装备槽当前没有已装备物品。"
        "bag full":
            return "背包已满，无法卸下装备。"
        "equipment not found":
            return "找不到该装备实例，请刷新后重试。"
        "equipment must be unequipped to repair":
            return "请先卸下装备后再修复。"
        "equipment is not damaged":
            return "该装备无需修复。"
        "insufficient repair materials":
            return "修复材料不足。"
        "invalid repair material item":
            return "修复材料无效。"
        "equipment operation failed":
            return "装备操作失败，请稍后再试。"
        "equipment repair failed":
            return "装备修复失败，请稍后再试。"
        "repair config missing":
            return "修复配置缺失，请联系管理员。"
        "player not found":
            return "角色数据异常，请重新登录。"
    if error_code == 20004:
        return "交互失败，请稍后再试。"
    if error_code == 50001:
        return "背包数据加载失败，请稍后再试。"
    if not message.strip_edges().is_empty():
        return message.strip_edges()
    return "操作失败，请稍后再试。"
