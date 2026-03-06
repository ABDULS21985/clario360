export function getAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("access_token");
}

export function getRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("refresh_token");
}

export function isAuthenticated(): boolean {
  const token = getAccessToken();
  if (!token) return false;

  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    return payload.exp * 1000 > Date.now();
  } catch {
    return false;
  }
}

export function getUserFromToken(): {
  uid: string;
  tid: string;
  email: string;
  roles: string[];
} | null {
  const token = getAccessToken();
  if (!token) return null;

  try {
    return JSON.parse(atob(token.split(".")[1]));
  } catch {
    return null;
  }
}

export function hasPermission(permission: string): boolean {
  const user = getUserFromToken();
  if (!user) return false;
  if (user.roles.includes("super_admin") || user.roles.includes("tenant_admin")) return true;
  return false;
}

export function clearAuth(): void {
  if (typeof window === "undefined") return;
  localStorage.removeItem("access_token");
  localStorage.removeItem("refresh_token");
  localStorage.removeItem("tenant_id");
}
