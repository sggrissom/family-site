export interface AuthCache {
    Id: number
    Email: string
    FirstName: string
    LastName: string
    isAdmin: boolean
}

let _auth: AuthCache | null = (() => {
  try {
    return JSON.parse(localStorage.getItem("auth-cache")!) as AuthCache;
  } catch {
    return null;
  }
})();

export function getAuth(): AuthCache | null {
  return _auth;
}

export function setAuth(a: AuthCache) {
  _auth = a;
  localStorage.setItem("auth-cache", JSON.stringify(a));
}

export function clearAuth() {
  _auth = null;
  localStorage.removeItem("auth-cache");
}
