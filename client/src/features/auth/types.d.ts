declare module 'AuthModels' {
  export interface User {
    accessToken: string;
    refreshToken: string;
  }

  export type AuthState = {
    isLoggedIn: boolean;
    user: User;
    error: string;
  };

  export type UserCredentials = {
    email: string;
    password: string;
  };

  export type UserRegister = {
    name: string;
    email: string;
    password: string;
    passwordConf: string;
  };
}
