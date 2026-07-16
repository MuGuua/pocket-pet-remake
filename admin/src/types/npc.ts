export interface AdminNPCEntitySummary {
  entity_id: number;
  entity_code: string;
  display_name: string;
  entity_type: number;
  scene_id: number;
  scene_name: string;
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
  display_name: string;
  entity_type: number;
  scene_id: number;
  status: number;
}

export interface AdminUpdateNPCEntityPayload extends AdminCreateNPCEntityPayload {}

export interface AdminWorldSceneSummary {
  scene_id: number;
  scene_code: string;
  scene_name: string;
  status: number;
}

export interface AdminWorldSceneListResult {
  items: AdminWorldSceneSummary[];
}

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
  battle_encounter_entity_id?: number;
  linked_quest_id?: number;
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
  conditions?: AdminDialogueConditions;
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
  battle_encounter_entity_id?: number;
  linked_quest_id?: number;
  conditions?: AdminDialogueConditions;
  status: number;
}

export interface AdminUpdateNPCMenuEntryPayload extends Omit<AdminCreateNPCMenuEntryPayload, 'entry_id'> {}

export interface AdminNPCDialogueFilters {
  entity_id?: string;
  entry_id?: string;
  status?: string;
}

export interface AdminDialogueConditions {
  quest_id?: number;
  quest_state?: string;
  objective_id?: number;
  objective_completed?: boolean;
}

export interface AdminDialogueEffectGrantItem {
  item_id?: number;
  quantity?: number;
}

export interface AdminDialogueEffects {
  notice?: string;
  quest_event?: string;
  accept_quest_id?: number;
  submit_quest_id?: number;
  grant_items?: AdminDialogueEffectGrantItem[];
}

export interface AdminNPCDialogueOption {
  option_id: string;
  option_text: string;
  option_format: string;
  next_node_id: string;
  sort_order: number;
  conditions?: AdminDialogueConditions;
}

export interface AdminNPCDialogueNode {
  node_id: string;
  node_type: string;
  speaker: string;
  content: string;
  content_format: string;
  portrait_key: string;
  next_node_id: string;
  client_animation_key: string;
  client_animation_block: boolean;
  sort_order: number;
  conditions?: AdminDialogueConditions;
  effects?: AdminDialogueEffects;
  options: AdminNPCDialogueOption[];
}

export interface AdminNPCDialogueSummary {
  entity_id: number;
  entry_id: string;
  dialogue_code: string;
  title: string;
  start_node_id: string;
  version: number;
  status: number;
  updated_at: string;
  created_at: string;
}

export interface AdminNPCDialogueListResult {
  items: AdminNPCDialogueSummary[];
  total: number;
  page: number;
  page_size: number;
}

export interface AdminNPCDialogueDetail extends AdminNPCDialogueSummary {
  nodes: AdminNPCDialogueNode[];
}

export interface AdminCreateNPCDialoguePayload {
  entity_id: number;
  entry_id: string;
  dialogue_code: string;
  title: string;
  start_node_id: string;
  version: number;
  status: number;
  nodes: AdminNPCDialogueNode[];
}

export interface AdminUpdateNPCDialoguePayload extends Omit<AdminCreateNPCDialoguePayload, 'entry_id'> {}
