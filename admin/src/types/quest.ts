export interface AdminQuestTemplateSummary {
  quest_id: number;
  name: string;
  quest_type: string;
  title: string;
  chapter: number;
  sort_order: number;
  accept_mode: string;
  submit_mode: string;
  auto_track: boolean;
  client_icon_id: number;
  min_player_level: number;
  status: number;
  status_text: string;
  updated_at: string;
  created_at: string;
}

export interface AdminQuestTemplateListResult {
  items: AdminQuestTemplateSummary[];
  total: number;
  page: number;
  page_size: number;
}

export interface AdminQuestObjectiveGuideInput {
  scene_id?: number;
  npc_id?: number;
  text?: string;
  menu_entry_id?: number;
  dialogue_entry_id?: number;
}

export interface AdminQuestObjectiveInput {
  objective_id: number;
  event_type: string;
  description: string;
  target_value: number;
  target_selector: Record<string, unknown>;
  guide?: AdminQuestObjectiveGuideInput;
}

export interface AdminQuestRewardInput {
  type: 'exp' | 'item' | 'gold';
  value: number;
  item_id: number;
  count: number;
}

export interface AdminQuestTemplateDetail {
  quest_id: number;
  name: string;
  quest_type: string;
  title: string;
  description: string;
  completion_prompt_text: string;
  chapter: number;
  sort_order: number;
  accept_mode: string;
  submit_mode: string;
  auto_track: boolean;
  client_icon_id: number;
  start_npc_id: number;
  submit_npc_id: number;
  accept_animation_key: string;
  submit_animation_key: string;
  min_player_level: number;
  status: number;
  status_text: string;
  pre_quest_ids: number[];
  objectives: AdminQuestObjectiveInput[];
  rewards: AdminQuestRewardInput[];
  created_at: string;
  updated_at: string;
}

export interface AdminQuestTemplateFilters {
  quest_id?: string;
  quest_type?: string;
  title?: string;
  status?: string;
}

export interface AdminCreateQuestTemplatePayload {
  quest_id?: number;
  name: string;
  quest_type: string;
  title: string;
  description: string;
  completion_prompt_text: string;
  chapter: number;
  sort_order: number;
  accept_mode: string;
  submit_mode: string;
  auto_track: boolean;
  client_icon_id: number;
  start_npc_id: number;
  submit_npc_id: number;
  accept_animation_key: string;
  submit_animation_key: string;
  min_player_level: number;
  status: number;
  pre_quest_ids: number[];
  objectives: AdminQuestObjectiveInput[];
  rewards: AdminQuestRewardInput[];
}

export interface AdminUpdateQuestTemplatePayload extends Omit<AdminCreateQuestTemplatePayload, 'quest_id'> {}

export interface AdminPlayerQuestObjectiveInput {
  objective_id: number;
  description: string;
  current_value: number;
  target_value: number;
  completed: boolean;
}

export interface AdminPlayerQuestSummary {
  record_id: number;
  player_id: number;
  player_name: string;
  quest_id: number;
  quest_title: string;
  quest_type: string;
  state: string;
  tracked: boolean;
  reward_claimed: boolean;
  accepted_at: string | null;
  completed_at: string | null;
  updated_at: string;
  created_at: string;
}

export interface AdminPlayerQuestListResult {
  items: AdminPlayerQuestSummary[];
  total: number;
  page: number;
  page_size: number;
}

export interface AdminPlayerQuestDetail {
  record_id: number;
  player_id: number;
  player_name: string;
  quest_id: number;
  quest_title: string;
  quest_type: string;
  state: string;
  tracked: boolean;
  reward_claimed: boolean;
  accepted_at: string | null;
  completed_at: string | null;
  submitted_at: string | null;
  created_at: string;
  updated_at: string;
  objectives: AdminPlayerQuestObjectiveInput[];
}

export interface AdminPlayerQuestFilters {
  record_id?: string;
  player_id?: string;
  quest_id?: string;
  state?: string;
  tracked?: string;
}

export interface AdminCreatePlayerQuestPayload {
  player_id: number;
  quest_id: number;
  state: string;
  tracked: boolean;
  reward_claimed: boolean;
  objectives: AdminPlayerQuestObjectiveInput[];
}

export interface AdminUpdatePlayerQuestPayload extends AdminCreatePlayerQuestPayload {}
