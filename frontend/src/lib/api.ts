import axios, { type AxiosInstance, type AxiosError, type InternalAxiosRequestConfig } from 'axios';

// API Configuration
const PLATFORM_API_URL = import.meta.env.PUBLIC_PLATFORM_API_URL || 'http://localhost:8080';
const INSTANCE_API_URL = import.meta.env.PUBLIC_INSTANCE_API_URL || 'http://localhost:8081';

// Create axios instances
const createApiClient = (baseURL: string): AxiosInstance => {
  const client = axios.create({
    baseURL,
    timeout: 30000,
    headers: {
      'Content-Type': 'application/json',
    },
  });

  // Request interceptor
  client.interceptors.request.use(
    (config: InternalAxiosRequestConfig) => {
      const token = localStorage.getItem('auth_token');
      if (token && config.headers) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      return config;
    },
    (error: AxiosError) => {
      return Promise.reject(error);
    }
  );

  // Response interceptor
  client.interceptors.response.use(
    (response) => response,
    (error: AxiosError) => {
      if (error.response?.status === 401) {
        localStorage.removeItem('auth_token');
        localStorage.removeItem('user');
        window.location.href = '/login';
      }
      return Promise.reject(error);
    }
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
      message: axiosError.response?.data?.message || axiosError.message || 'An error occurred',
      code: axiosError.response?.data?.code,
      details: axiosError.response?.data?.details,
    };
  }
  return {
    message: error instanceof Error ? error.message : 'An unexpected error occurred',
  };
};

// Request helper with error handling
export const apiRequest = async <T>(
  client: AxiosInstance,
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE',
  url: string,
  data?: unknown
): Promise<T> => {
  try {
    const response = await client.request<T>({
      method,
      url,
      data,
    });
    return response.data;
  } catch (error) {
    throw handleApiError(error);
  }
};

export default {
  platform: platformApi,
  instance: instanceApi,
};
