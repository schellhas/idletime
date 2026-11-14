import { http } from "./http";
import type { User } from "./user";

export const register = (username: string, email: string, password: string) =>
  http.post("/auth/register", { username, email, password });

export const login = (identifier: string, password: string) =>
  http.post<User>("/auth/login", { identifier, password });

export const logout = () => http.post("/auth/logout");

export const verifyEmail = (token: string) => http.post("/auth/verify-email", { token });

export const resendVerification = (email: string) =>
  http.post("/auth/resend-verification", { email });

export const forgotPassword = (email: string) => http.post("/auth/forgot-password", { email });

export const resetPassword = (token: string, password: string) =>
  http.post("/auth/reset-password", { token, password });
