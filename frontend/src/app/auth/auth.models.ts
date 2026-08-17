export type UserRole = 'coordinator' | 'caregiver';

export interface AuthUser {
  id: string;
  first_name: string;
  last_name: string;
  email: string;
  role: UserRole;
}

export interface LoginResponse {
  token: string;
  user: AuthUser;
}
