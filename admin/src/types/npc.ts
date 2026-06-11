export interface AdminNPCEntitySummary {
  entity_id: number;
  entity_code: string;
  display_name: string;
  entity_type: number;
  scene_id: number;
  pos_x: number;
  pos_y: number;
  dir: number;
  speed: number;
  status: number;
  status_text: string;
  updated_at: string;
  created_at: string;
}

export interface AdminNPCEntityListResult {
  items: AdminNPCEntitySummary[];
  total: number;
  page: number;
  page_size: number;
}

export interface AdminNPCEntityDetail extends AdminNPCEntitySummary {}

export interface AdminNPCEntityFilters {
  entity_id?: string;
  scene_id?: string;
  entity_type?: string;
  status?: string;
  name?: string;
}

export interface AdminCreateNPCEntityPayload {
  entity_id: number;
  entity_code: string;
  display_name: string;
  entity_type: number;
  scene_id: number;
  pos_x: number;
  pos_y: number;
  dir: number;
  speed: number;
  status: number;
}

export interface AdminUpdateNPCEntityPayload extends Omit<AdminCreateNPCEntityPayload, 'entity_id'> {}

export interface AdminNPCMenuEntrySummary {
  entity_id: number;
  entry_id: string;
  entry_type: string;
  title: string;
  subtitle: string;
  state: string;
  priority: number;
  sort_order: number;
  action_result_type: string;
  status: number;
  status_text: string;
  updated_at: string;
  created_at: string;
}

export interface AdminNPCMenuEntryListResult {
  items: AdminNPCMenuEntrySummary[];
  total: number;
  page: number;
  page_size: number;
}

export interface AdminNPCMenuEntryDetail extends AdminNPCMenuEntrySummary {
  action_notice: string;
}

export interface AdminNPCMenuEntryFilters {
  entity_id?: string;
  entry_id?: string;
  status?: string;
}

export interface AdminCreateNPCMenuEntryPayload {
  entity_id: number;
  entry_id: string;
  entry_type: string;
  title: string;
  subtitle: string;
  state: string;
  priority: number;
  sort_order: number;
  action_result_type: string;
  action_notice: string;
  status: number;
}

export interface AdminUpdateNPCMenuEntryPayload extends Omit<AdminCreateNPCMenuEntryPayload, 'entry_id'> {}
