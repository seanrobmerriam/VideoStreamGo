import axios, {
  type AxiosInstance,
  type AxiosError,
  type InternalAxiosRequestConfig,
} from "axios";

// API Configuration
const PLATFORM_API_URL =
  import.meta.env.PUBLIC_PLATFORM_API_URL || "http://localhost:8080";
const INSTANCE_API_URL =
  import.meta.env.PUBLIC_INSTANCE_API_URL || "http://localhost:8081";

// Create axios instances
const createApiClient = (baseURL: string): AxiosInstance => {
  const client = axios.create({
    baseURL,
    timeout: 30000,
    withCredentials: true, // Required for httpOnly cookies - browser sends cookies automatically
    headers: {
      "Content-Type": "application/json",
    },
  });

  // Request interceptor for auth token
  client.interceptors.request.use((config) => {
    const token = localStorage.getItem("token");
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  });

  // Response interceptor
  client.interceptors.response.use(
    (response) => response,
    (error: AxiosError) => {
      if (error.response?.status === 401) {
        const publicPaths = [
          "/",
          "/auth/login",
          "/auth/register",
          "/auth/forgot-password",
          "/auth/reset-password",
        ];
        const currentPath = window.location.pathname.replace(/\/$/, "") || "/";
        if (!publicPaths.includes(currentPath)) {
          window.location.href = "/auth/login";
        }
      }
      return Promise.reject(error);
    },
  );

  return client;
};

// API Clients
export const platformApi = createApiClient(PLATFORM_API_URL);
export const instanceApi = createApiClient(INSTANCE_API_URL);

// Types
export interface ApiResponse<T> {
  data: T;
  message?: string;
  success: boolean;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface ErrorResponse {
  message: string;
  code?: string;
  details?: Record<string, string>;
}

// Helper function to handle API errors
export const handleApiError = (error: unknown): ErrorResponse => {
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<ErrorResponse>;
    return {
      message:
        axiosError.response?.data?.message ||
        axiosError.message ||
        "An error occurred",
      code: axiosError.response?.data?.code,
      details: axiosError.response?.data?.details,
    };
  }
  return {
    message:
      error instanceof Error ? error.message : "An unexpected error occurred",
  };
};

// Request helper with error handling
export const apiRequest = async <T>(
  client: AxiosInstance,
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE",
  url: string,
  data?: unknown,
): Promise<T> => {
  try {
    const response = await client.request<T>({
      method,
      url,
      data,
    });
    return response.data;
  } catch (error) {
    const errorResponse = handleApiError(error);
    throw new Error(errorResponse.message);
  }
};

export default {
  platform: platformApi,
  instance: instanceApi,
};
