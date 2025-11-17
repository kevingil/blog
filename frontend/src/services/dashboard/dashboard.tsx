import { atom, useAtomValue, useSetAtom } from 'jotai';
import { createContext, useContext, ReactNode } from 'react';

// Jotai atom for page title
export const pageTitleAtom = atom<string>("Dashboard");

// AdminDashboard Context type definition
export interface AdminDashboardContext {
  pageTitle: string;
  setPageTitle: (title: string) => void;
}

// Create the AdminDashboard Context
const AdminDashboardContext = createContext<AdminDashboardContext | null>(null);

// AdminDashboard Provider Props
interface AdminDashboardProviderProps {
  children: ReactNode;
}

// AdminDashboard Provider component
export function AdminDashboardProvider({ children }: AdminDashboardProviderProps) {
  const pageTitle = useAtomValue(pageTitleAtom);
  const setPageTitle = useSetAtom(pageTitleAtom);

  return (
    <AdminDashboardContext.Provider value={{ pageTitle, setPageTitle }}>
      {children}
    </AdminDashboardContext.Provider>
  );
}

// Hook to use the admin dashboard context
export function useAdminDashboard(): AdminDashboardContext {
  const context = useContext(AdminDashboardContext);
  if (!context) {
    throw new Error('useAdminDashboard must be used within an AdminDashboardProvider');
  }
  return context;
}

