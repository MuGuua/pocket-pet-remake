import { formatSkillReferences, type SkillReferenceMap } from '../utils/skillReference';

interface SkillReferenceTextProps {
  skillIds: number[];
  map: SkillReferenceMap;
  emptyText?: string;
}

// SkillReferenceText 统一把 skill_id 列表渲染为技能名称文案。
export function SkillReferenceText({ skillIds, map, emptyText = '-' }: SkillReferenceTextProps) {
  return <>{formatSkillReferences(skillIds, map, emptyText)}</>;
}
