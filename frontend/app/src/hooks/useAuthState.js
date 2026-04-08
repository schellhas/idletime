import { useEffect, useState } from 'react';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080';

async function apiFetch(path, options = {}) {
  const { body, headers, ...rest } = options;
  const finalHeaders = {
    Accept: 'application/json',
    ...headers,
  };

  if (body !== undefined) {
    finalHeaders['Content-Type'] = 'application/json';
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    credentials: 'include',
    headers: finalHeaders,
    body,
    ...rest,
  });

  const raw = await response.text();
  let data = {};

  if (raw) {
    try {
      data = JSON.parse(raw);
    } catch {
      data = { raw };
    }
  }

  if (!response.ok) {
    throw new Error(data.error || data.message || `Request failed (${response.status})`);
  }

  return data;
}

export function useAuthState() {
  const [bootstrapping, setBootstrapping] = useState(true);
  const [user, setUser] = useState(null);
  const [authMode, setAuthMode] = useState('login');
  const [statusMessage, setStatusMessage] = useState('');
  const [errorMessage, setErrorMessage] = useState('');
  const [verificationLink, setVerificationLink] = useState('');
  const [accountPanelOpen, setAccountPanelOpen] = useState(false);
  const [loginForm, setLoginForm] = useState({ identifier: '', password: '' });
  const [registerForm, setRegisterForm] = useState({ username: '', email: '', password: '' });

  async function bootstrapSession() {
    try {
      const userData = await apiFetch('/auth/me');
      setUser(userData.user ?? null);
    } catch {
      setUser(null);
    } finally {
      setBootstrapping(false);
    }
  }

  async function handleLogin(event) {
    event.preventDefault();
    setErrorMessage('');
    setStatusMessage('Signing in...');

    try {
      const response = await apiFetch('/auth/login', {
        method: 'POST',
        body: JSON.stringify(loginForm),
      });

      setUser(response.user);
      setStatusMessage('');
      setLoginForm({ identifier: '', password: '' });
      setVerificationLink('');
      setAccountPanelOpen(false);
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function handleRegister(event) {
    event.preventDefault();
    setErrorMessage('');
    setStatusMessage('Creating account...');

    try {
      const response = await apiFetch('/auth/register', {
        method: 'POST',
        body: JSON.stringify(registerForm),
      });

      setVerificationLink(response.verification_url ?? '');
      setRegisterForm({ username: '', email: '', password: '' });
      setAuthMode('login');
      setStatusMessage(response.message);
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  async function handleLogout() {
    setErrorMessage('');

    try {
      await apiFetch('/auth/logout', { method: 'POST' });
      setUser(null);
      setStatusMessage('Signed out.');
      setAccountPanelOpen(false);
    } catch (error) {
      setErrorMessage(error.message);
    }
  }

  useEffect(() => {
    void bootstrapSession();
  }, []);

  return {
    bootstrapping,
    user,
    setUser,
    authMode,
    setAuthMode,
    statusMessage,
    setStatusMessage,
    errorMessage,
    setErrorMessage,
    verificationLink,
    setVerificationLink,
    accountPanelOpen,
    setAccountPanelOpen,
    loginForm,
    setLoginForm,
    registerForm,
    setRegisterForm,
    handleLogin,
    handleRegister,
    handleLogout,
  };
}
