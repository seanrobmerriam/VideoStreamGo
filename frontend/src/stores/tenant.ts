import { atom, map } from 'nanostores';

// Tenant Types
export interface TenantConfig {
  id: string;
  name: string;
  subdomain: string;
  logo?: string;
  favicon?: string;
  primaryColor: string;
  secondaryColor: string;
  customDomain?: string;
}

export interface TenantState {
  currentTenant: TenantConfig | null;
  tenants: TenantConfig[];
  isLoading: boolean;
  error: string | null;
}

// Tenant Store
export const $tenant = map<TenantState>({
  currentTenant: null,
  tenants: [],
  isLoading: false,
  error: null,
});

// Computed values
export const $currentTenant = atom<TenantConfig | null>($tenant.get().currentTenant);
export const $tenants = atom<TenantConfig[]>($tenant.get().tenants);
export const $tenantLoading = atom<boolean>($tenant.get().isLoading);
export const $tenantError = atom<string | null>($tenant.get().error);

// Subscribe to changes
$tenant.subscribe((state) => {
  $currentTenant.set(state.currentTenant);
  $tenants.set(state.tenants);
  $tenantLoading.set(state.isLoading);
  $tenantError.set(state.error);
});

// Actions
export const setCurrentTenant = (tenant: TenantConfig | null): void => {
  $tenant.setKey('currentTenant', tenant);
  if (tenant) {
    applyTenantBranding(tenant);
  }
};

export const setTenants = (tenants: TenantConfig[]): void => {
  $tenant.setKey('tenants', tenants);
};

export const setTenantLoading = (isLoading: boolean): void => {
  $tenant.setKey('isLoading', isLoading);
};

export const setTenantError = (error: string | null): void => {
  $tenant.setKey('error', error);
};

// Apply tenant branding to CSS variables
const applyTenantBranding = (tenant: TenantConfig): void => {
  const root = document.documentElement;
  
  if (tenant.primaryColor) {
    root.style.setProperty('--color-primary', hexToRgb(tenant.primaryColor));
  }
  if (tenant.secondaryColor) {
    root.style.setProperty('--color-secondary', hexToRgb(tenant.secondaryColor));
  }
};

const hexToRgb = (hex: string): string => {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
  if (!result) return '14 165 233';
  
  const r = result[1] ?? '';
  const g = result[2] ?? '';
  const b = result[3] ?? '';
  return `${parseInt(r, 16)} ${parseInt(g, 16)} ${parseInt(b, 16)}`;
};

// Load tenant from subdomain
export const loadTenantFromSubdomain = async (subdomain: string): Promise<void> => {
  setTenantLoading(true);
  setTenantError(null);
  
  try {
    const { apiRequest, platformApi } = await import('../lib/api');
    const tenant = await apiRequest<TenantConfig>(platformApi, 'GET', `/api/v1/instances/by-subdomain/${subdomain}`);
    setCurrentTenant(tenant);
  } catch (error) {
    setTenantError(error instanceof Error ? error.message : 'Failed to load tenant');
  } finally {
    setTenantLoading(false);
  }
};

// Initialize tenant from URL
export const initializeTenant = (): void => {
  const hostname = window.location.hostname;
  const subdomain = hostname.split('.')[0];
  
  if (hostname === 'localhost' || hostname.includes('127.0.0.1')) {
    setCurrentTenant({
      id: 'default',
      name: 'VideoStreamGo',
      subdomain: 'demo',
      primaryColor: '#0ea5e9',
      secondaryColor: '#64748b',
    });
    return;
  }
  
  loadTenantFromSubdomain(subdomain ?? '');
};
