import { createContext, useContext, ReactNode, useState, useEffect } from 'react';

interface User {
  id: string;
  name: string;
  email: string;
}

interface UserContextType {
  user: User | null;
}

const UserContext = createContext<UserContextType>({ user: null });

export function UserProvider({ children, userPromise }: { children: ReactNode; userPromise: Promise<User | null> }) {
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    userPromise.then(setUser);
  }, [userPromise]);

  return (
    <UserContext.Provider value={{ user }}>
      {children}
    </UserContext.Provider>
  );
}

export function useUser() {
  return useContext(UserContext);
} 
