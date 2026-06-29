import { fetchAllAdminSkillDefinitions } from '../services/skillDefinition';

// SkillReferenceMap 缓存 skill_id 与 skill_name 的双向映射，供后台技能展示与录入。
export interface SkillReferenceMap {
  idToName: Record<number, string>;
  nameToId: Record<string, number>;
}

const emptySkillReferenceMap: SkillReferenceMap = {
  idToName: {},
  nameToId: {},
};

// loadSkillReferenceMap 分页拉取全部系统技能模板，构建 id/name 映射表。
export async function loadSkillReferenceMap(): Promise<SkillReferenceMap> {
  const items = await fetchAllAdminSkillDefinitions();
  const idToName: Record<number, string> = {};
  const nameToId: Record<string, number> = {};
  for (const item of items) {
    idToName[item.skill_id] = item.skill_name;
    nameToId[item.skill_name.trim()] = item.skill_id;
  }
  return { idToName, nameToId };
}

// formatSkillReferences 将 skill_id 列表格式化为技能名称展示文案。
export function formatSkillReferences(skillIds: number[], map: SkillReferenceMap, emptyText = '-'): string {
  if (skillIds.length === 0) {
    return emptyText;
  }
  return skillIds
    .map((skillID) => map.idToName[skillID] ?? `未知技能(${skillID})`)
    .join('、');
}

// formatSkillReferenceInput 将 skill_id 列表转换为表单中的技能名称输入串。
export function formatSkillReferenceInput(skillIds: number[], map: SkillReferenceMap): string {
  if (skillIds.length === 0) {
    return '';
  }
  return skillIds
    .map((skillID) => map.idToName[skillID] ?? String(skillID))
    .join(',');
}

// parseSkillReferenceInput 解析运营填写的技能名称（兼容仍输入数字 ID 的情况）。
export function parseSkillReferenceInput(raw: string, map: SkillReferenceMap): number[] {
  const parts = raw.split(',').map((item) => item.trim()).filter(Boolean);
  const result: number[] = [];
  const seen = new Set<number>();
  for (const part of parts) {
    const numericValue = Number(part);
    let skillID = 0;
    if (Number.isInteger(numericValue) && numericValue > 0 && map.idToName[numericValue]) {
      skillID = numericValue;
    } else if (map.nameToId[part]) {
      skillID = map.nameToId[part];
    }
    if (skillID > 0 && !seen.has(skillID)) {
      seen.add(skillID);
      result.push(skillID);
    }
  }
  return result;
}

export { emptySkillReferenceMap };
