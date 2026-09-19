import { beforeEach, describe, expect, it, vi } from 'vitest'

const { apiMock } = vi.hoisted(() => ({
  apiMock: {
    get: vi.fn(),
    post: vi.fn(),
  },
}))

vi.mock('../core/api', () => ({
  api: apiMock,
  previewMode: false,
}))

import { login, register } from '../core/session'

const user = {
  id: 1,
  username: 'turnstile-test',
  email: 'test@example.com',
  role: 'user' as const,
}

describe('Smirel auth Turnstile contract', () => {
  beforeEach(() => {
    apiMock.get.mockReset()
    apiMock.post.mockReset()
    localStorage.clear()
  })

  it('sends the Turnstile token with password login', async () => {
    apiMock.post.mockResolvedValueOnce({
      data: {
        access_token: 'login-access-token',
        user,
      },
    })

    await login('test@example.com', 'secret', 'turnstile-login-token')

    expect(apiMock.post).toHaveBeenCalledWith('/auth/login', {
      email: 'test@example.com',
      password: 'secret',
      turnstile_token: 'turnstile-login-token',
    })
  })

  it('sends the Turnstile token with registration', async () => {
    apiMock.post.mockResolvedValueOnce({
      data: {
        access_token: 'register-access-token',
        user,
      },
    })

    await register('test@example.com', 'secret', 'turnstile-register-token')

    expect(apiMock.post).toHaveBeenCalledWith('/auth/register', {
      email: 'test@example.com',
      password: 'secret',
      turnstile_token: 'turnstile-register-token',
    })
  })
})