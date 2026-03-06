"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth-store";
import { isAuthenticated, getUserFromToken } from "@/lib/auth";
import type { User } from "@/types/models";

export function useAuth() {
  const router = useRouter();
  const { user, setUser, setAuthenticated, setLoading, logout } = useAuthStore();

  useEffect(() => {
    if (isAuthenticated()) {
      const tokenUser = getUserFromToken();
      if (tokenUser) {
        const u: User = {
          id: tokenUser.uid,
          tenant_id: tokenUser.tid,
          email: tokenUser.email,
          first_name: "",
          last_name: "",
          is_active: true,
          roles: tokenUser.roles,
          created_at: "",
          updated_at: "",
        };
        setUser(u);
      }
      setAuthenticated(true);
    } else {
      setAuthenticated(false);
    }
    setLoading(false);
  }, [setUser, setAuthenticated, setLoading]);

  const handleLogout = () => {
    logout();
    router.push("/login");
  };

  return { user, isAuthenticated: useAuthStore((s) => s.isAuthenticated), isLoading: useAuthStore((s) => s.isLoading), logout: handleLogout };
}
