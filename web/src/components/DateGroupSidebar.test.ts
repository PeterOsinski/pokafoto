import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'

const { mockApiGet, mockRouteQuery, mockPush, mockReplace } = vi.hoisted(() => ({
  mockApiGet: vi.fn(),
  mockRouteQuery: {} as Record<string, string>,
  mockPush: vi.fn(),
  mockReplace: vi.fn(),
}))

const mockRoute = { query: mockRouteQuery }

vi.mock('../api/client', () => ({
  default: {
    get: mockApiGet,
  },
}))

vi.mock('vue-router', () => ({
  useRoute: () => mockRoute,
  useRouter: () => ({
    push: mockPush,
    replace: mockReplace,
  }),
}))

import DateGroupSidebar from './DateGroupSidebar.vue'

function makeTimelineGroup(period: string, label: string, count: number) {
  return { period, label, count, thumbnailUrl: '', startDate: `${period}-01T00:00:00Z`, endDate: '' }
}

describe('DateGroupSidebar', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.keys(mockRouteQuery).forEach((k) => delete mockRouteQuery[k])
  })

  it('renders loading state initially', async () => {
    mockApiGet.mockImplementation(() => new Promise(() => {}))
    const wrapper = mount(DateGroupSidebar)
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Loading')
  })

  it('renders empty state when no date groups returned', async () => {
    mockApiGet.mockResolvedValue({ data: { groups: [] } })
    const wrapper = mount(DateGroupSidebar)
    await new Promise((r) => setTimeout(r, 10))
    expect(wrapper.text()).toContain('No photos with dates')
  })

  it('renders error state on API failure', async () => {
    mockApiGet.mockRejectedValue(new Error('Network error'))
    const wrapper = mount(DateGroupSidebar)
    await new Promise((r) => setTimeout(r, 10))
    expect(wrapper.text()).toContain('Failed to load dates')
  })

  it('renders year and month groups from timeline data', async () => {
    mockApiGet.mockResolvedValue({
      data: {
        groups: [
          makeTimelineGroup('2026-06', 'June 2026', 15),
          makeTimelineGroup('2026-05', 'May 2026', 10),
          makeTimelineGroup('2025-12', 'December 2025', 5),
        ],
      },
    })
    const wrapper = mount(DateGroupSidebar)
    await new Promise((r) => setTimeout(r, 10))

    expect(wrapper.text()).toContain('2026')
    expect(wrapper.text()).toContain('2025')
    expect(wrapper.text()).toContain('All photos')
  })

  it('calls timeline API on mount', async () => {
    mockApiGet.mockResolvedValue({ data: { groups: [] } })
    mount(DateGroupSidebar)
    await new Promise((r) => setTimeout(r, 10))
    expect(mockApiGet).toHaveBeenCalledWith('/timeline', { params: { granularity: 'month' } })
  })

  it('highlights active month based on date_from query param', async () => {
    mockApiGet.mockResolvedValue({
      data: {
        groups: [
          makeTimelineGroup('2026-06', 'June 2026', 15),
          makeTimelineGroup('2026-05', 'May 2026', 10),
        ],
      },
    })
    mockRouteQuery.date_from = '2026-06-01'
    mockRouteQuery.date_to = '2026-07-01'

    const wrapper = mount(DateGroupSidebar)
    await new Promise((r) => setTimeout(r, 10))

    const yearBtn = wrapper.findAll('button').find(b => b.text().includes('2026'))
    expect(yearBtn).toBeTruthy()
  })
})
