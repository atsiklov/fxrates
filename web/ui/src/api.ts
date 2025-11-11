type ErrorResponse = {
  error?: string
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
  })

  const contentType = response.headers.get('content-type') ?? ''
  const isJSON = contentType.includes('application/json')
  const payload = isJSON ? await response.json() : await response.text()

  if (!response.ok) {
    const message =
      typeof payload === 'string'
        ? payload || 'Request failed'
        : (payload as ErrorResponse)?.error ?? 'Request failed'
    throw new Error(message)
  }

  return payload as T
}

export type SupportedCurrenciesResponse = {
  codes: string[]
}

export type RateResponse = {
  base: string
  quote: string
  value: number
  updated_at: string
}

export type ScheduleUpdateResponse = {
  update_id: string
}

export type RateUpdateStatus = 'pending' | 'applied' | string

export type RateUpdatePendingResponse = {
  update_id: string
  base: string
  quote: string
  status: RateUpdateStatus
}

export type RateUpdateAppliedResponse = RateUpdatePendingResponse & {
  value: number
  updated_at: string
}

const API_PATHS = {
  supported: '/api/v1/rates/supported-currencies',
  latest: (base: string, quote: string) => `/api/v1/rates/${base}/${quote}`,
  schedule: '/api/v1/rates/updates',
  update: (id: string) => `/api/v1/rates/updates/${id}`,
}

export async function fetchSupportedCurrencies() {
  const res = await request<SupportedCurrenciesResponse>(API_PATHS.supported, {
    method: 'GET',
  })
  return res.codes
}

export type RateView = {
  base: string
  quote: string
  value: number
  updatedAt: string
}

export async function fetchLatestRate(base: string, quote: string): Promise<RateView> {
  const res = await request<RateResponse>(API_PATHS.latest(base, quote), { method: 'GET' })
  return {
    base: res.base,
    quote: res.quote,
    value: res.value,
    updatedAt: res.updated_at,
  }
}

export async function scheduleRateUpdate(base: string, quote: string) {
  const res = await request<ScheduleUpdateResponse>(API_PATHS.schedule, {
    method: 'POST',
    body: JSON.stringify({ base, quote }),
  })
  return {
    updateId: res.update_id,
  }
}

export type RateUpdateView = {
  updateId: string
  base: string
  quote: string
  status: RateUpdateStatus
  value?: number
  updatedAt?: string
}

export async function fetchRateUpdate(updateId: string): Promise<RateUpdateView> {
  const res = await request<RateUpdatePendingResponse | RateUpdateAppliedResponse>(API_PATHS.update(updateId), {
    method: 'GET',
  })

  if ('value' in res && 'updated_at' in res) {
    return {
      updateId: res.update_id,
      base: res.base,
      quote: res.quote,
      status: res.status,
      value: res.value,
      updatedAt: res.updated_at,
    }
  }

  return {
    updateId: res.update_id,
    base: res.base,
    quote: res.quote,
    status: res.status,
  }
}
