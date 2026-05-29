import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import AdminView from './AdminView.vue'
import api from '../api/client'

vi.mock('@/api/client', () => ({
  default: {
    post: vi.fn(),
    get: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('axios', () => ({
  default: {
    create: () => ({
      interceptors: { request: { use: vi.fn() }, response: { use: vi.fn() } },
    }),
  },
}))

const mockStatsResponse = {
  total_files: 150,
  total_size_bytes: 1073741824,
  cache_size_bytes: 524288000,
  disk_total_bytes: 10737418240,
  disk_free_bytes: 5368709120,
  disk_used_bytes: 5368709120,
  disk_utilization_pct: 50.0,
  users: [
    {
      id: 'user-1',
      username: 'admin',
      role: 'admin',
      file_count: 100,
      total_size_bytes: 524288000,
    },
    {
      id: 'user-2',
      username: 'member1',
      role: 'member',
      file_count: 50,
      total_size_bytes: 549453824,
    },
  ],
}

const mockWorkersResponse = {
  queue_length: 3,
  active_workers: 2,
  total_workers: 4,
  processing_jobs: [
    {
      job_id: 'job-1',
      filename: 'photo.jpg',
      status: 'processing',
      stage: 'thumbnails',
      progress: 0.75,
    },
  ],
  completed_total: 42,
  failed_total: 1,
  skipped_total: 5,
}

const mockBreakdownResponse = {
  media_types: [
    { media_type: 'photo', count: 120, size_bytes: 800000000 },
    { media_type: 'video', count: 25, size_bytes: 2500000000 },
    { media_type: 'file', count: 5, size_bytes: 50000000 },
  ],
  extensions: [
    { extension: 'jpeg', count: 80, size_bytes: 500000000 },
    { extension: 'png', count: 40, size_bytes: 300000000 },
    { extension: 'mp4', count: 25, size_bytes: 2500000000 },
    { extension: 'pdf', count: 5, size_bytes: 50000000 },
  ],
  total_size: 3350000000,
}

describe('AdminView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  function mountAdmin() {
    return mount(AdminView)
  }

  it('loads and displays stats on mount', async () => {
    const mockGet = api.get as ReturnType<typeof vi.fn>
    mockGet.mockImplementation((url: string) => {
      if (url === '/admin/stats') return Promise.resolve({ data: mockStatsResponse })
      if (url === '/admin/workers') return Promise.resolve({ data: mockWorkersResponse })
      if (url === '/admin/files/breakdown') return Promise.resolve({ data: mockBreakdownResponse })
      return Promise.resolve({ data: [] })
    })

    const wrapper = mountAdmin()
    await flushPromises()

    expect(wrapper.text()).toContain('Storage')
    expect(wrapper.text()).toContain('5.0 GB')
    expect(wrapper.text()).toContain('150')
  })

  it('loads and displays worker pool on mount', async () => {
    const mockGet = api.get as ReturnType<typeof vi.fn>
    mockGet.mockImplementation((url: string) => {
      if (url === '/admin/stats') return Promise.resolve({ data: mockStatsResponse })
      if (url === '/admin/workers') return Promise.resolve({ data: mockWorkersResponse })
      if (url === '/admin/files/breakdown') return Promise.resolve({ data: mockBreakdownResponse })
      return Promise.resolve({ data: [] })
    })

    const wrapper = mountAdmin()
    await flushPromises()

    expect(wrapper.text()).toContain('Worker Pool')
    expect(wrapper.text()).toContain('2 / 4')
    expect(wrapper.text()).toContain('42')
    expect(wrapper.text()).toContain('1')
  })

  it('displays processing job details', async () => {
    const mockGet = api.get as ReturnType<typeof vi.fn>
    mockGet.mockImplementation((url: string) => {
      if (url === '/admin/stats') return Promise.resolve({ data: mockStatsResponse })
      if (url === '/admin/workers') return Promise.resolve({ data: mockWorkersResponse })
      if (url === '/admin/files/breakdown') return Promise.resolve({ data: mockBreakdownResponse })
      return Promise.resolve({ data: [] })
    })

    const wrapper = mountAdmin()
    await flushPromises()

    expect(wrapper.text()).toContain('photo.jpg')
    expect(wrapper.text()).toContain('75%')
    expect(wrapper.text()).toContain('thumbnails')
  })

  it('polls stats every 10 seconds', async () => {
    const mockGet = api.get as ReturnType<typeof vi.fn>
    mockGet.mockImplementation((url: string) => {
      if (url === '/admin/stats') return Promise.resolve({ data: mockStatsResponse })
      if (url === '/admin/workers') return Promise.resolve({ data: mockWorkersResponse })
      if (url === '/admin/files/breakdown') return Promise.resolve({ data: mockBreakdownResponse })
      return Promise.resolve({ data: [] })
    })

    mountAdmin()
    await flushPromises()

    const statsCalls = mockGet.mock.calls.filter((c: string[]) => c[0] === '/admin/stats').length
    expect(statsCalls).toBe(1)

    vi.advanceTimersByTime(10000)
    await flushPromises()

    const updatedCalls = mockGet.mock.calls.filter((c: string[]) => c[0] === '/admin/stats').length
    expect(updatedCalls).toBe(2)
  })

  it('polls workers every 5 seconds', async () => {
    const mockGet = api.get as ReturnType<typeof vi.fn>
    mockGet.mockImplementation((url: string) => {
      if (url === '/admin/stats') return Promise.resolve({ data: mockStatsResponse })
      if (url === '/admin/workers') return Promise.resolve({ data: mockWorkersResponse })
      if (url === '/admin/files/breakdown') return Promise.resolve({ data: mockBreakdownResponse })
      return Promise.resolve({ data: [] })
    })

    mountAdmin()
    await flushPromises()

    vi.advanceTimersByTime(5000)
    await flushPromises()

    const workerCalls = mockGet.mock.calls.filter((c: string[]) => c[0] === '/admin/workers').length
    expect(workerCalls).toBe(2)
  })

  it('loads and displays file breakdown', async () => {
    const mockGet = api.get as ReturnType<typeof vi.fn>
    mockGet.mockImplementation((url: string) => {
      if (url === '/admin/stats') return Promise.resolve({ data: mockStatsResponse })
      if (url === '/admin/workers') return Promise.resolve({ data: mockWorkersResponse })
      if (url === '/admin/files/breakdown') return Promise.resolve({ data: mockBreakdownResponse })
      return Promise.resolve({ data: [] })
    })

    const wrapper = mountAdmin()
    await flushPromises()

    expect(wrapper.text()).toContain('File Breakdown')
    expect(wrapper.text()).toContain('By Media Type')
    expect(wrapper.text()).toContain('By Extension')
    expect(wrapper.text()).toContain('Total Size (all files)')
    expect(wrapper.text()).toContain('photo')
    expect(wrapper.text()).toContain('video')
    expect(wrapper.text()).toContain('file')
    expect(wrapper.text()).toContain('jpeg')
    expect(wrapper.text()).toContain('mp4')
    expect(wrapper.text()).toContain('pdf')
    expect(wrapper.text()).toContain('120')
    expect(wrapper.text()).toContain('25')
    expect(wrapper.text()).toContain('762.9 MB')
    expect(wrapper.text()).toContain('3.1 GB')
  })
})
