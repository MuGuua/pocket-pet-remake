import { Typography } from 'antd';

interface AdminItemIconProps {
  icon?: string;
  size?: number;
  fallbackText?: string;
}

/** 后台物品 icon 展示；res:// 路径无法直接在 Web 预览时显示占位符。 */
export function AdminItemIcon({ icon = '', size = 32, fallbackText = '物' }: AdminItemIconProps) {
  const imageSrc = toAdminImageSrc(icon);
  if (!imageSrc) {
    return (
      <span
        style={{
          width: size,
          height: size,
          display: 'inline-grid',
          placeItems: 'center',
          borderRadius: 6,
          background: '#f5f5f5',
          border: '1px solid #d9d9d9',
          fontSize: Math.max(12, Math.floor(size * 0.5)),
        }}
      >
        {fallbackText}
      </span>
    );
  }
  return <img src={imageSrc} alt="" style={{ width: size, height: size, objectFit: 'contain' }} />;
}

function toAdminImageSrc(icon: string): string {
  const trimmedIcon = icon.trim();
  if (trimmedIcon === '' || trimmedIcon.startsWith('res://')) {
    return '';
  }
  return trimmedIcon;
}

interface ItemMentionChipProps {
  itemID: number;
  itemName?: string;
  icon?: string;
}

/** 后台预览区使用的物品 mention 胶囊。 */
export function ItemMentionChip({ itemID, itemName, icon }: ItemMentionChipProps) {
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, verticalAlign: 'middle' }}>
      <AdminItemIcon icon={icon ?? ''} size={20} />
      <Typography.Text strong>{itemName ?? `物品${itemID}`}</Typography.Text>
    </span>
  );
}

interface PetMentionChipProps {
  petID: number;
  petName?: string;
}

/** 后台预览区使用的宠物 mention 胶囊。 */
export function PetMentionChip({ petID, petName }: PetMentionChipProps) {
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, verticalAlign: 'middle' }}>
      <AdminItemIcon icon="" size={20} fallbackText="宠" />
      <Typography.Text strong>{petName ?? `宠物${petID}`}</Typography.Text>
    </span>
  );
}
