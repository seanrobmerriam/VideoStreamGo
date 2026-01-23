import { apiRequest, platformApi, instanceApi, type ApiResponse } from './api';

// Types
export interface User {
  id: string;
  email: string;
  name: string;
  role: 'admin' | 'customer' | 'user';
  avatar?: string;
  createdAt: string;
}

export interface LoginCredentials {
  email: string;
  password: string;
}

export interface RegisterData {
  email: string;
  password: string;
  name: string;
  company?: string;
}

export interface AuthResponse {
  user: User;
  token: string;
}

// Platform Auth
export const platformAuth = {
  async login(credentials: LoginCredentials): Promise<AuthResponse> {
    return apiRequest<AuthResponse>(platformApi, 'POST', '/api/v1/auth/login', credentials);
  },

  async register(data: RegisterData): Promise<AuthResponse> {
    return apiRequest<AuthResponse>(platformApi, 'POST', '/api/v1/auth/register', data);
  },

  async getCurrentUser(): Promise<User> {
    return apiRequest<User>(platformApi, 'GET', '/api/v1/auth/me');
  },

  async logout(): Promise<void> {
    await apiRequest<void>(platformApi, 'POST', '/api/v1/auth/logout');
  },

  async updateProfile(data: Partial<User>): Promise<User> {
    return apiRequest<User>(platformApi, 'PUT', '/api/v1/auth/profile', data);
  },

  async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    await apiRequest<void>(platformApi, 'PUT', '/api/v1/auth/password', {
      currentPassword,
      newPassword,
    });
  },
};

// Instance Auth
export const instanceAuth = {
  async login(credentials: LoginCredentials, subdomain: string): Promise<AuthResponse> {
    return apiRequest<AuthResponse>(instanceApi, 'POST', `/${subdomain}/api/v1/auth/login`, credentials);
  },

  async register(data: RegisterData & { subdomain: string }): Promise<AuthResponse> {
    return apiRequest<AuthResponse>(instanceApi, 'POST', `/${data.subdomain}/api/v1/auth/register`, data);
  },

  async getCurrentUser(subdomain: string): Promise<User> {
    return apiRequest<User>(instanceApi, 'GET', `/${subdomain}/api/v1/auth/me`);
  },
};

// Token management
export const tokenManager = {
  getToken(): string | null {
    return localStorage.getItem('auth_token');
  },

  setToken(token: string): void {
    localStorage.setItem('auth_token', token);
  },

  removeToken(): void {
    localStorage.removeItem('auth_token');
  },

  getStoredUser(): User | null {
    const user = localStorage.getItem('user');
    return user ? JSON.parse(user) : null;
  },

  setStoredUser(user: User): void {
    localStorage.setItem('user', JSON.stringify(user));
  },

  removeStoredUser(): void {
    localStorage.removeItem('user');
  },

  clearAll(): void {
    this.removeToken();
    this.removeStoredUser();
  },
};

// Auth helper
export const isAuthenticated = (): boolean => {
  return !!tokenManager.getToken();
};

export const hasRole = (user: User | null, roles: User['role'][]): boolean => {
  return user ? roles.includes(user.role) : false;
};

export const isAdmin = (user: User | null): boolean => {
  return user?.role === 'admin';
};

export const isCustomer = (user: User | null): boolean => {
  return user?.role === 'customer';
};
