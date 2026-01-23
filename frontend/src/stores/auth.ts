import { atom, map } from 'nanostores';
import type { User } from '../lib/auth';
import { tokenManager, isAuthenticated as checkAuth } from '../lib/auth';

// Auth State
export interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
}

// Auth Store
export const $auth = map<AuthState>({
  user: null,
  isAuthenticated: false,
  isLoading: true,
  error: null,
});

// Computed values
export const $user = atom<User | null>($auth.get().user);
export const $isAuthenticated = atom<boolean>($auth.get().isAuthenticated);
export const $isLoading = atom<boolean>($auth.get().isLoading);
export const $authError = atom<string | null>($auth.get().error);

// Subscribe to changes
$auth.subscribe((state) => {
  $user.set(state.user);
  $isAuthenticated.set(state.isAuthenticated);
  $isLoading.set(state.isLoading);
  $authError.set(state.error);
});

// Actions
export const setUser = (user: User | null): void => {
  $auth.setKey('user', user);
  $auth.setKey('isAuthenticated', !!user);
  if (user) {
    tokenManager.setStoredUser(user);
  }
};

export const setLoading = (isLoading: boolean): void => {
  $auth.setKey('isLoading', isLoading);
};

export const setError = (error: string | null): void => {
  $auth.setKey('error', error);
};

export const initializeAuth = (): void => {
  const token = tokenManager.getToken();
  const user = tokenManager.getStoredUser();
  
  if (token && user) {
    setUser(user);
  } else {
    $auth.setKey('isLoading', false);
  }
};

export const login = async (email: string, password: string): Promise<void> => {
  setLoading(true);
  setError(null);
  
  try {
    const { platformAuth } = await import('../lib/auth');
    const response = await platformAuth.login({ email, password });
    tokenManager.setToken(response.token);
    tokenManager.setStoredUser(response.user);
    setUser(response.user);
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Login failed';
    setError(message);
    throw error;
  } finally {
    setLoading(false);
  }
};

export const register = async (data: {
  email: string;
  password: string;
  name: string;
  company?: string;
}): Promise<void> => {
  setLoading(true);
  setError(null);
  
  try {
    const { platformAuth } = await import('../lib/auth');
    const response = await platformAuth.register(data);
    tokenManager.setToken(response.token);
    tokenManager.setStoredUser(response.user);
    setUser(response.user);
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Registration failed';
    setError(message);
    throw error;
  } finally {
    setLoading(false);
  }
};

export const logout = async (): Promise<void> => {
  setLoading(true);
  
  try {
    const { platformAuth } = await import('../lib/auth');
    await platformAuth.logout();
  } catch {
    // Continue with logout even if API call fails
  } finally {
    tokenManager.clearAll();
    setUser(null);
    setLoading(false);
  }
};

export const clearError = (): void => {
  setError(null);
};
