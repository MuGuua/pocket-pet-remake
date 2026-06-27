-- 064_enhance_material_category.sql
-- 明确强化材料子分类 item_sub_type=equipment_enhance，供背包筛选与强化消耗校验使用。

COMMENT ON COLUMN item_definition.item_sub_type IS
  '物品子分类；equipment_enhance 表示强化材料，客户端 category=enhance_material 与服务端 EnhancePreview.materials 均按此筛选';
