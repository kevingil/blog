// types/models.ts
declare module 'ProfileModels' {
  
    export interface Role {
      id: number;
      name: 'ADMIN' | 'EDITOR';
      created_at: string;
      updated_at: string;
    }
  
    export interface UserProfile {
      id: number;
      name: string;
      email: string;
      avatar: string | null;
      role_id: number;
      role?: Role;
      created_at: string;
      updated_at: string;
    }
  
    export interface UserProfileState {
      profile: UserProfile | null;
      loading: boolean;
      error: string | null;
      updateSuccess: boolean;
    }
  
    export interface UpdateProfileData {
      name?: string;
      email?: string;
      avatar?: string;
      role_id?: number;
    }
  }
