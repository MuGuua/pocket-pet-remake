import type React from 'react';
import { bbcodeToHtml } from './richTextBbcode';

/** 物品占位符 token。 */
export const ITEM_MENTION_TOKEN_PATTERN = /\{item:(\d+)\}/g;
/** 宠物占位符 token。 */
export const PET_MENTION_TOKEN_PATTERN = /\{pet:(\d+)\}/g;
/** 玩家名占位符（仅剧情对白使用）。 */
export const PLAYER_NAME_TOKEN = '{player_name}';

export interface ItemMentionPreviewRecord {
  item_id: number;
  item_name: string;
  icon?: string;
}

export interface PetMentionPreviewRecord {
  pet_id: number;
  pet_name: string;
}

/** 生成物品 mention 占位符。 */
export function buildItemMentionToken(itemID: number): string {
  return `{item:${itemID}}`;
}

/** 生成宠物 mention 占位符。 */
export function buildPetMentionToken(petID: number): string {
  return `{pet:${petID}}`;
}

/** 从文案中提取去重后的物品 ID。 */
export function extractMentionItemIDs(content: string): number[] {
  const ids = new Set<number>();
  for (const match of content.matchAll(ITEM_MENTION_TOKEN_PATTERN)) {
    const itemID = Number(match[1]);
    if (itemID > 0) {
      ids.add(itemID);
    }
  }
  return Array.from(ids);
}

/** 从文案中提取去重后的宠物 ID。 */
export function extractMentionPetIDs(content: string): number[] {
  const ids = new Set<number>();
  for (const match of content.matchAll(PET_MENTION_TOKEN_PATTERN)) {
    const petID = Number(match[1]);
    if (petID > 0) {
      ids.add(petID);
    }
  }
  return Array.from(ids);
}

/** 将 BBCode + 物品/宠物/玩家名占位符渲染为后台预览片段。 */
export function renderMentionContentFragments(
  content: string,
  itemMap: Record<number, ItemMentionPreviewRecord>,
  petMap: Record<number, PetMentionPreviewRecord>,
  options?: { showPlayerName?: boolean },
): React.ReactNode[] {
  const tokenPattern = /(\{player_name\}|\{item:(\d+)\}|\{pet:(\d+)\})/g;
  const fragments: React.ReactNode[] = [];
  let lastIndex = 0;
  let match: RegExpExecArray | null = tokenPattern.exec(content);
  while (match) {
    if (match.index > lastIndex) {
      const textSlice = content.slice(lastIndex, match.index);
      fragments.push(
        <span
          key={`text-${lastIndex}`}
          data-rich-source-start={lastIndex}
          data-rich-source-end={match.index}
          data-rich-source-kind="text"
          dangerouslySetInnerHTML={{ __html: bbcodeToHtml(textSlice) }}
        />,
      );
    }
    if (match[1] === PLAYER_NAME_TOKEN && options?.showPlayerName) {
      fragments.push(
        <strong
          key={`player-${match.index}`}
          data-rich-source-start={match.index}
          data-rich-source-end={match.index + match[0].length}
          data-rich-source-kind="mention"
        >
          玩家
        </strong>,
      );
    } else if (match[2]) {
      const itemID = Number(match[2]);
      const item = itemMap[itemID];
      fragments.push(
        <strong
          key={`item-${match.index}-${itemID}`}
          data-rich-source-start={match.index}
          data-rich-source-end={match.index + match[0].length}
          data-rich-source-kind="mention"
          style={{ color: '#f0d5b1' }}
        >
          {item?.item_name ?? `物品${itemID}`}
        </strong>,
      );
    } else if (match[3]) {
      const petID = Number(match[3]);
      const pet = petMap[petID];
      fragments.push(
        <strong
          key={`pet-${match.index}-${petID}`}
          data-rich-source-start={match.index}
          data-rich-source-end={match.index + match[0].length}
          data-rich-source-kind="mention"
          style={{ color: '#f0d5b1' }}
        >
          {pet?.pet_name ?? `宠物${petID}`}
        </strong>,
      );
    }
    lastIndex = match.index + match[0].length;
    match = tokenPattern.exec(content);
  }
  if (lastIndex < content.length) {
    fragments.push(
      <span
        key={`text-${lastIndex}`}
        data-rich-source-start={lastIndex}
        data-rich-source-end={content.length}
        data-rich-source-kind="text"
        dangerouslySetInnerHTML={{ __html: bbcodeToHtml(content.slice(lastIndex)) }}
      />,
    );
  }
  return fragments;
}
