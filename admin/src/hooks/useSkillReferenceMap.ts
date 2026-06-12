import { useEffect, useState } from 'react';
import { emptySkillReferenceMap, loadSkillReferenceMap, type SkillReferenceMap } from '../utils/skillReference';

// useSkillReferenceMap 在后台页面挂载时加载技能名称映射，供技能展示与录入复用。
export function useSkillReferenceMap() {
  const [map, setMap] = useState<SkillReferenceMap>(emptySkillReferenceMap);
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    let cancelled = false;
    loadSkillReferenceMap()
      .then((nextMap) => {
        if (!cancelled) {
          setMap(nextMap);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return { map, loading };
}
