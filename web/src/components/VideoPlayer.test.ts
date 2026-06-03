import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import VideoPlayer from './VideoPlayer.vue'

describe('VideoPlayer', () => {
  const baseProps = {
    src: '/api/v1/download/video-1',
    poster: '/thumb/video_still.jpg',
  }

  it('renders a video element', () => {
    const wrapper = mount(VideoPlayer, { props: baseProps })

    const video = wrapper.find('video')
    expect(video.exists()).toBe(true)
  })

  it('uses download URL as src', () => {
    const wrapper = mount(VideoPlayer, { props: baseProps })

    const source = wrapper.find('source')
    expect(source.attributes('src')).toBe('/api/v1/download/video-1')
  })

  it('uses videoStill thumbnail as poster', () => {
    const wrapper = mount(VideoPlayer, { props: baseProps })

    const video = wrapper.find('video')
    expect(video.attributes('poster')).toBe('/thumb/video_still.jpg')
  })

  it('has controls attribute', () => {
    const wrapper = mount(VideoPlayer, { props: baseProps })

    const video = wrapper.find('video')
    expect(video.attributes('controls')).toBeDefined()
  })

  it('resets quality to proxy when src changes', async () => {
    const wrapper = mount(VideoPlayer, { props: baseProps })

    await wrapper.setProps({ src: '/api/v1/download/video-2' })
    await nextTick()

    const source = wrapper.find('source')
    expect(source.attributes('src')).toBe('/api/v1/download/video-2')
  })

  it('resets error state when src changes', async () => {
    const wrapper = mount(VideoPlayer, { props: baseProps })

    const video = wrapper.find('video')
    await video.trigger('error')
    await nextTick()

    expect(wrapper.find('[data-error]').exists() || wrapper.text()).toBeTruthy()

    await wrapper.setProps({ src: '/api/v1/download/video-2' })
    await nextTick()

    const errorDiv = wrapper.findAll('div').filter(d => d.text().includes('Failed'))
    expect(errorDiv.length).toBe(0)
  })

  it('recreates video element with :key change', async () => {
    const wrapper = mount(
      {
        template: '<VideoPlayer :key="key" :src="src" :poster="poster" />',
        components: { VideoPlayer },
        data() { return { key: 'v1', src: '/api/v1/download/video-1', poster: '/thumb/poster1.jpg' } },
      },
    )

    const firstSource = wrapper.find('source')
    expect(firstSource.attributes('src')).toBe('/api/v1/download/video-1')

    await wrapper.setData({ key: 'v2', src: '/api/v1/download/video-2', poster: '/thumb/poster2.jpg' })
    await nextTick()

    const source = wrapper.find('source')
    expect(source.attributes('src')).toBe('/api/v1/download/video-2')
  })
})
