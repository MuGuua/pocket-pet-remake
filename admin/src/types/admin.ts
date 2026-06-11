export interface AdminLoginResult {
  admin_user_id: number;
  display_name: string;
  role_keys: string[];
  access_token: string;
  expire_at: number;
}

export interface AdminSessionProfile {
  admin_user_id: number;
  account_name: string;
  display_name: string;
  role_keys: string[];
  permissions: string[];
}
