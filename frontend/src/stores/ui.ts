import { atom, map } from 'nanostores';

// UI State Types
export interface Notification {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title: string;
  message?: string;
  duration?: number;
}

export interface ModalState {
  isOpen: boolean;
  component: string | null;
  props?: Record<string, unknown>;
}

export interface UIState {
  theme: 'light' | 'dark' | 'system';
  sidebarOpen: boolean;
  notifications: Notification[];
  modals: ModalState[];
  isMobileMenuOpen: boolean;
}

// UI Store
export const $ui = map<UIState>({
  theme: 'system',
  sidebarOpen: true,
  notifications: [],
  modals: [],
  isMobileMenuOpen: false,
});

// Computed values
export const $theme = atom<'light' | 'dark' | 'system'>($ui.get().theme);
export const $sidebarOpen = atom<boolean>($ui.get().sidebarOpen);
export const $notifications = atom<Notification[]>($ui.get().notifications);
export const $isMobileMenuOpen = atom<boolean>($ui.get().isMobileMenuOpen);

// Subscribe to changes
$ui.subscribe((state) => {
  $theme.set(state.theme);
  $sidebarOpen.set(state.sidebarOpen);
  $notifications.set(state.notifications);
  $isMobileMenuOpen.set(state.isMobileMenuOpen);
});

// Theme Actions
export const setTheme = (theme: 'light' | 'dark' | 'system'): void => {
  $ui.setKey('theme', theme);
  applyTheme(theme);
};

export const toggleTheme = (): void => {
  const current = $ui.get().theme;
  const next = current === 'light' ? 'dark' : current === 'dark' ? 'system' : 'light';
  setTheme(next);
};

const applyTheme = (theme: 'light' | 'dark' | 'system'): void => {
  const root = document.documentElement;
  
  if (theme === 'system') {
    const systemDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    root.classList.toggle('dark', systemDark);
  } else {
    root.classList.toggle('dark', theme === 'dark');
  }
  
  localStorage.setItem('theme', theme);
};

export const initializeTheme = (): void => {
  const saved = localStorage.getItem('theme') as 'light' | 'dark' | 'system' | null;
  const theme = saved || 'system';
  setTheme(theme);
  
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if ($ui.get().theme === 'system') {
      applyTheme('system');
    }
  });
};

// Sidebar Actions
export const toggleSidebar = (): void => {
  $ui.setKey('sidebarOpen', !$ui.get().sidebarOpen);
};

export const setSidebarOpen = (open: boolean): void => {
  $ui.setKey('sidebarOpen', open);
};

// Mobile Menu Actions
export const toggleMobileMenu = (): void => {
  $ui.setKey('isMobileMenuOpen', !$ui.get().isMobileMenuOpen);
};

export const setMobileMenuOpen = (open: boolean): void => {
  $ui.setKey('isMobileMenuOpen', open);
};

// Notification Actions
export const addNotification = (notification: Omit<Notification, 'id'>): string => {
  const id = crypto.randomUUID();
  const newNotification: Notification = {
    ...notification,
    id,
    duration: notification.duration ?? 5000,
  };
  
  $ui.setKey('notifications', [...$ui.get().notifications, newNotification]);
  
  if (newNotification.duration && newNotification.duration > 0) {
    setTimeout(() => {
      removeNotification(id);
    }, newNotification.duration);
  }
  
  return id;
};

export const removeNotification = (id: string): void => {
  $ui.setKey(
    'notifications',
    $ui.get().notifications.filter((n) => n.id !== id)
  );
};

export const clearNotifications = (): void => {
  $ui.setKey('notifications', []);
};

// Notification helpers
export const notifySuccess = (title: string, message?: string): string => {
  return addNotification({ type: 'success', title, message });
};

export const notifyError = (title: string, message?: string): string => {
  return addNotification({ type: 'error', title, message, duration: 7000 });
};

export const notifyWarning = (title: string, message?: string): string => {
  return addNotification({ type: 'warning', title, message });
};

export const notifyInfo = (title: string, message?: string): string => {
  return addNotification({ type: 'info', title, message });
};

// Modal Actions
export const openModal = (component: string, props?: Record<string, unknown>): void => {
  $ui.setKey('modals', [...$ui.get().modals, { isOpen: true, component, props }]);
};

export const closeModal = (): void => {
  const modals = $ui.get().modals;
  if (modals.length > 0) {
    const newModals = modals.slice(0, -1);
    $ui.setKey('modals', newModals);
  }
};

export const closeAllModals = (): void => {
  $ui.setKey('modals', []);
};

export const isModalOpen = (component: string): boolean => {
  return $ui.get().modals.some((m) => m.component === component);
};
