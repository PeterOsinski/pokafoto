import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import FolderPasswordDialog from './FolderPasswordDialog.vue'

vi.mock('../api/client', () => {
  const api = {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  }
  return { default: api }
})

describe('FolderPasswordDialog', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    document.body.innerHTML = ''
  })

  function mountDialog(overrides: Record<string, any> = {}) {
    const wrapper = mount(FolderPasswordDialog, {
      props: {
        visible: true,
        folderId: 'folder-1',
        mode: 'unlock' as const,
        ...overrides,
      },
      attachTo: document.body,
    })
    return { wrapper, getText: () => document.body.textContent || '' }
  }

  it('shows custom prompt when passwordHint is set', () => {
    const { getText } = mountDialog({ passwordHint: 'Enter your favorite color' })
    expect(getText()).toContain('Enter your favorite color')
    expect(getText()).not.toContain('This folder is password-protected')
  })

  it('shows generic message when passwordHint is empty', () => {
    const { getText } = mountDialog({ passwordHint: '' })
    expect(getText()).toContain('This folder is password-protected')
  })

  it('shows generic message when passwordHint is undefined', () => {
    const { getText } = mountDialog({ passwordHint: undefined })
    expect(getText()).toContain('This folder is password-protected')
  })

  it('disables unlock button when password is empty', () => {
    const { wrapper } = mountDialog({ passwordHint: 'Test' })
    const buttons = document.querySelectorAll('button')
    const unlockBtn = Array.from(buttons).find(b => b.textContent === 'Unlock')
    expect(unlockBtn).toBeTruthy()
    expect(unlockBtn!.hasAttribute('disabled')).toBe(true)
    wrapper.unmount()
  })

  it('shows Set Folder Password title in set mode', () => {
    const { getText } = mountDialog({ mode: 'set' })
    expect(getText()).toContain('Set Folder Password')
  })

  it('shows Folder Password title in status mode', () => {
    const { getText } = mountDialog({ mode: 'status' })
    expect(getText()).toContain('Folder Password')
  })
})
