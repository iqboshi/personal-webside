import type { AdminSession, Article, Checkin, SiteContent, SiteData, TodoDayStatus, TodoTaskDraft } from '../types'
import { requestJSON } from './request'

const ADMIN_TOKEN_KEY = 'mengqing-homepage-admin-token'

interface CheckinListResponse {
  items: Checkin[]
}

export async function fetchCheckins(): Promise<Checkin[]> {
  const response = await requestJSON<CheckinListResponse>('/api/checkins')
  return response.items
}

export function fetchTodayTodos(date?: string): Promise<TodoDayStatus> {
  const query = date ? `?date=${encodeURIComponent(date)}` : ''
  return requestJSON<TodoDayStatus>(`/api/todos/today${query}`)
}

export function fetchAdminSession(): Promise<AdminSession> {
  return requestJSON<AdminSession>('/api/admin/session', {
    headers: authHeaders(),
  })
}

export async function loginAdmin(username: string, password: string): Promise<AdminSession> {
  const session = await requestJSON<AdminSession>('/api/admin/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  })
  if (session.token) saveAdminToken(session.token)
  return session
}

export async function logoutAdmin(): Promise<AdminSession> {
  try {
    return await requestJSON<AdminSession>('/api/admin/logout', {
      method: 'POST',
      headers: authHeaders(),
    })
  } finally {
    clearAdminToken()
  }
}

export async function saveCheckin(date: string, note: string): Promise<Checkin> {
  return requestJSON<Checkin>(`/api/admin/checkins/${encodeURIComponent(date)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify({ note }),
  })
}

export function saveTodoTasks(tasks: TodoTaskDraft[], date?: string): Promise<TodoDayStatus> {
  return requestJSON<TodoDayStatus>('/api/admin/todos/tasks', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify({ date, tasks }),
  })
}

export function setTodoCompletion(date: string, taskId: string, done: boolean): Promise<TodoDayStatus> {
  return requestJSON<TodoDayStatus>('/api/admin/todos/complete', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify({ date, taskId, done }),
  })
}

export function fetchAdminContent(): Promise<SiteContent> {
  return requestJSON<SiteContent>('/api/admin/content', {
    headers: authHeaders(),
  })
}

export function saveAdminContent(profile: SiteData, articles: Article[]): Promise<SiteContent> {
  return requestJSON<SiteContent>('/api/admin/content', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify({ profile, articles }),
  })
}

function authHeaders(): Record<string, string> {
  const token = readAdminToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}

function readAdminToken(): string {
  try {
    return window.localStorage.getItem(ADMIN_TOKEN_KEY) || ''
  } catch {
    return ''
  }
}

function saveAdminToken(token: string) {
  try {
    window.localStorage.setItem(ADMIN_TOKEN_KEY, token)
  } catch {
    // Token storage is a convenience for static cross-domain hosting.
  }
}

function clearAdminToken() {
  try {
    window.localStorage.removeItem(ADMIN_TOKEN_KEY)
  } catch {
    // Ignore storage failures; the server session is still cleared when cookies work.
  }
}
