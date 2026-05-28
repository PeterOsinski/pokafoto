import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
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
})
