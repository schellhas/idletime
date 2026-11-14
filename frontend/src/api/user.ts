import { http } from "./http";

export interface User {
  id: number;
  username: string;
  email: string;
  emailVerified: boolean;
  createdAt: string;
}

export const EXPORT_URL = "/api/v1/me/export";

export const getCurrentUser = () => http.get<User>("/me");

export const updateUsername = (username: string) => http.patch<User>("/me", { username });

export const changePassword = (currentPassword: string, newPassword: string) =>
  http.post("/me/password", { currentPassword, newPassword });

export const deleteAccount = (password: string) => http.delete("/me", { password });
