import axios from 'axios'

// Use relative URLs to go through nginx proxy
const API_URL = import.meta.env.VITE_API_URL || '/api'
const WS_URL = import.meta.env.VITE_WS_URL || (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host + '/ws'

const api = axios.create({
  baseURL: API_URL,
  withCredentials: true,
})

export interface User {
  user_id: string
  username: string
  name?: string
  is_admin: boolean
}

export interface Conversation {
  id: string
  user_id: string
  title: string
  status: string
  created_at: string
  updated_at: string
  message_count?: number
}

export interface Message {
  id: string
  conversation_id: string
  role: string
  content: string
  timestamp: string
}

export const auth = {
  login: async (username: string, password: string) => {
    const { data } = await api.post<{ user_id: string; username: string; is_admin: boolean; token: string }>('/auth/login', {
      username,
      password,
    })
    return data
  },

  logout: async () => {
    await api.post('/auth/logout')
  },

  me: async () => {
    const { data } = await api.get<User>('/auth/me')
    return data
  },
}

export const conversations = {
  list: async () => {
    const { data } = await api.get<Conversation[]>('/conversations')
    return data
  },

  get: async (id: string) => {
    const { data } = await api.get<Conversation>(`/conversations/${id}`)
    return data
  },

  create: async (title?: string) => {
    const { data } = await api.post<Conversation>('/conversations', { title })
    return data
  },

  updateStatus: async (id: string, status: string) => {
    await api.patch(`/conversations/${id}`, { status })
  },

  delete: async (id: string) => {
    await api.delete(`/conversations/${id}`)
  },

  getLastActive: async () => {
    const { data } = await api.get<Conversation | null>('/conversations/last-active')
    return data
  },

  getMessages: async (id: string) => {
    const { data } = await api.get<Message[]>(`/conversations/${id}/messages`)
    return data
  },
}

export const messages = {
  list: async (conversationId: string) => {
    const { data } = await api.get<Message[]>(`/conversations/${conversationId}/messages`)
    return data
  },
  
  save: async (conversationId: string, role: string, content: string) => {
    const { data } = await api.post(`/conversations/${conversationId}/messages`, {
      role,
      content,
    })
    return data
  },
}

export interface Voice {
  id: string
  name: string
  description: string
  prompt: string
  voice_description: string
  hume_voice_id: string
  hume_config_id: string
  evi_version: string
  language_model_provider: string
  language_model_resource: string
  temperature: number
  created_at: string
  updated_at: string
}

export interface CreateVoiceRequest {
  name: string
  description?: string
  prompt: string
  voice_description?: string
  evi_version?: string
  temperature?: number
}

export const voices = {
  list: async () => {
    const { data } = await api.get<Voice[]>('/voices')
    return data
  },

  get: async (id: string) => {
    const { data } = await api.get<Voice>(`/voices/${id}`)
    return data
  },

  create: async (voice: CreateVoiceRequest) => {
    const { data } = await api.post<Voice>('/admin/voices', voice)
    return data
  },

  update: async (id: string, voice: Partial<CreateVoiceRequest>) => {
    const { data } = await api.patch<Voice>(`/admin/voices/${id}`, voice)
    return data
  },

  delete: async (id: string) => {
    await api.delete(`/admin/voices/${id}`)
  },

  sync: async (id: string) => {
    const { data } = await api.post<{ status: string; voice_id: string }>(`/admin/voices/${id}/sync`)
    return data
  },

  syncAll: async () => {
    const { data } = await api.post<{ status: string; results: Array<{ voice_id: string; name: string; status: string; error?: string; reason?: string }> }>('/admin/voices/sync')
    return data
  },
}

export interface AdminUser {
  id: string
  username: string
  name?: string
  is_admin: boolean
  created_at: string
}

export interface CreateUserRequest {
  username: string
  password: string
  name?: string
  is_admin: boolean
}

export interface UpdateUserRequest {
  password?: string
  name?: string
  is_admin?: boolean
}

export const users = {
  list: async () => {
    const { data } = await api.get<AdminUser[]>('/admin/users')
    return data
  },

  create: async (user: CreateUserRequest) => {
    const { data } = await api.post<AdminUser>('/admin/users', user)
    return data
  },

  update: async (id: string, user: UpdateUserRequest) => {
    const { data } = await api.patch<AdminUser>(`/admin/users/${id}`, user)
    return data
  },

  delete: async (id: string) => {
    await api.delete(`/admin/users/${id}`)
  },
}

export { WS_URL }

