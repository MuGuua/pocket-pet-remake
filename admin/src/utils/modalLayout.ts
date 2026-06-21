import type { CSSProperties } from 'react';

export const FIXED_FORM_MODAL_TOP = 32;

export const FIXED_FORM_MODAL_BODY_STYLE: CSSProperties = {
  height: 'min(680px, calc(100vh - 220px))',
  overflowY: 'auto',
  paddingRight: 16,
};

export const FIXED_FORM_MODAL_STYLES: { body: CSSProperties } = {
  body: FIXED_FORM_MODAL_BODY_STYLE,
};
