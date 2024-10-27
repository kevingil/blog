import http from './http';
import TokenService from './token.service';
///////////////// ajeitar /////////
class AuthService {
  async login(email: string, password: string) {
    return http
      .post('/auth/login', {
        email,
        password
      })
      .then((response) => {
        if (response.data.accessToken) {
          TokenService.setUser(response.data);
        }

        return response.data;
      });
  }

  logout() {
    TokenService.removeUser();
  }

  async register(name: string, email: string, password: string) {
    return http
      .post('/auth/signup', {
        name,
        email,
        password
      })
      .then((response) => {
        return response.data;
      });
  }
}

export default new AuthService();
