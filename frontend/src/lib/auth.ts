import { apiRequest, platformApi, instanceApi, type ApiResponse } from "./api";

// Types
export interface User {
  id: string;
  email: string;
  display_name: string; // ← was 'name'
  role: "admin" | "customer" | "user";
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
  token: string;
  expires_at: string;
  user: User;
}

/**
 * SECURITY FIX: Token Storage
 *
 * Tokens are now stored in httpOnly cookies instead of localStorage.
 *
 * BACKEND REQUIREMENT:
 * The server MUST set the following cookie header on login/register responses:
 *
 * ```
 * Set-Cookie: auth_token=<token>; HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=<expiry>
 * ```
 *
 * Benefits of httpOnly cookies:
 * - Cannot be accessed by JavaScript (prevents XSS token theft)
 * - Automatically sent with requests (withCredentials: true required)
 * - Protected by SameSite=Strict CSRF prevention
 *
 * IMPORTANT: The frontend uses withCredentials: true in axios config,
 * which enables automatic cookie transmission. The server must handle
 * cookie creation and invalidation.
 */

// Platform Auth
export const platformAuth = {
  async login(credentials: LoginCredentials): Promise<AuthResponse> {
    return apiRequest<AuthResponse>(
      platformApi,
      "POST",
      "/api/v1/auth/admin/login",
      credentials,
    );
  },

  async register(data: RegisterData): Promise<AuthResponse> {
    return apiRequest<AuthResponse>(
      platformApi,
      "POST",
      "/api/v1/auth/admin/register",
      {
        email: data.email,
        password: data.password,
        display_name: data.name,
      },
    );
  },

  async getCurrentUser(): Promise<User> {
    return apiRequest<User>(platformApi, "GET", "/api/v1/admin/me");
  },

  async logout(): Promise<void> {
    await apiRequest<void>(platformApi, "POST", "/api/v1/auth/logout");
  },

  async updateProfile(data: Partial<User>): Promise<User> {
    return apiRequest<User>(platformApi, "PUT", "/api/v1/auth/profile", data);
  },

  async changePassword(
    currentPassword: string,
    newPassword: string,
  ): Promise<void> {
    await apiRequest<void>(platformApi, "PUT", "/api/v1/auth/password", {
      currentPassword,
      newPassword,
    });
  },
};

// Instance Auth
export const instanceAuth = {
  async login(
    credentials: LoginCredentials,
    subdomain: string,
  ): Promise<AuthResponse> {
    return apiRequest<AuthResponse>(
      instanceApi,
      "POST",
      `/${subdomain}/api/v1/auth/admin/login`,
      credentials,
    );
  },

  async register(
    data: RegisterData & { subdomain: string },
  ): Promise<AuthResponse> {
    return apiRequest<AuthResponse>(
      instanceApi,
      "POST",
      `/${data.subdomain}/api/v1/auth/admin/register`,
      data,
    );
  },

  async getCurrentUser(subdomain: string): Promise<User> {
    return apiRequest<User>(instanceApi, "GET", `/${subdomain}/api/v1/auth/me`);
  },
};

// Token management - Cookie-based
// Note: We cannot read httpOnly cookies from JavaScript for security.
// The browser automatically sends them with requests when withCredentials: true is set.
export const tokenManager = {
  /**
   * Get token from cookie (non-httpOnly for display purposes only)
   * For actual auth, cookies are sent automatically - we can't read httpOnly cookies.
   * This is intentionally returns null as httpOnly cookies cannot be accessed by JS.
   */
  getToken(): string | null {
    // httpOnly cookies cannot be read by JavaScript - this is the security feature
    // The browser automatically sends cookies with requests
    // Return null to indicate we rely on cookie-based auth
    return null;
  },

  /**
   * Set token - NO OP for httpOnly cookies
   * The server sets the cookie via Set-Cookie header
   */
  setToken(_token: string): void {
    // Token is set by server via httpOnly cookie
    // This is a no-op for security - we cannot set httpOnly cookies from JavaScript
    console.warn("Token setting is handled server-side via httpOnly cookies");
  },

  /**
   * Remove token - NO OP for httpOnly cookies
   * The server should invalidate the cookie via logout endpoint
   */
  removeToken(): void {
    // Token removal is handled by server via logout endpoint
    // This is a no-op for security
    console.warn("Token removal is handled server-side via logout");
  },

  /**
   * Store user data in localStorage (non-sensitive data only)
   * User data is not sensitive - it's public profile info
   */
  getStoredUser(): User | null {
    const user = localStorage.getItem("user");
    return user ? JSON.parse(user) : null;
  },

  setStoredUser(user: User): void {
    // Store user in localStorage for UI display purposes
    // This is not sensitive auth data - just user profile info
    localStorage.setItem("user", JSON.stringify(user));
  },

  removeStoredUser(): void {
    localStorage.removeItem("user");
  },

  clearAll(): void {
    // For httpOnly cookies, we can't clear the token from JS
    // The logout API call handles cookie invalidation
    this.removeStoredUser();
  },
};

// Auth helper
export const isAuthenticated = (): boolean => {
  // With httpOnly cookies, we can't check the token directly
  // The auth store should call getCurrentUser() to verify authentication
  // This function now returns false - actual auth check is done server-side
  return false;
};

export const hasRole = (user: User | null, roles: User["role"][]): boolean => {
  return user ? roles.includes(user.role) : false;
};

export const isAdmin = (user: User | null): boolean => {
  return user?.role === "admin";
};

export const isCustomer = (user: User | null): boolean => {
  return user?.role === "customer";
};
