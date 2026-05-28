# 1. Executive Summary

## Product Name
**Drive** — Self-Hosted Photo & File Backup with Gallery

## Elevator Pitch
Drive is a self-hosted Google Photos alternative that gives you full control over your media. It combines automatic photo organization, lightning-fast gallery browsing, EXIF-powered GPS map visualization, and general file backup — all running on your own hardware via Docker.

## Target Audience
- Privacy-conscious individuals who want to own their data
- Home server / NAS owners (Synology, QNAP, Raspberry Pi, NUC)
- Families wanting a shared, self-hosted photo library
- Anyone migrating away from Google Photos / iCloud

## Key Differentiators
| Feature | Drive | Google Photos | PhotoPrism | Immich |
|---|---|---|---|---|
| Self-hosted, single binary | ✅ | ❌ | ✅ | ✅ |
| SQLite (zero-dependency DB) | ✅ | ❌ | ❌ (MariaDB) | ❌ (PostgreSQL) |
| S3-compatible storage | ✅ | ❌ | ❌ | ❌ |
| Local thumbnail cache tier | ✅ | ❌ | ❌ | ❌ |
| GPS map with photo clustering | ✅ | ✅ | ✅ | ✅ |
| General file backup (not just photos) | ✅ | ❌ | ❌ | ❌ |
| Go + Vue.js stack | ✅ | ❌ | ❌ | ❌ |

## High-Level Architecture
```
┌──────────────────────────────────────────────────────┐
│                    Docker Container                    │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐  │
│  │  Go Backend  │  │  Vue.js SPA  │  │   SQLite    │  │
│  │  (API + Jobs)│  │  (Gallery UI) │  │  (Metadata) │  │
│  └──────┬───────┘  └──────────────┘  └─────────────┘  │
│         │                                               │
│  ┌──────┴───────┐  ┌────────────────┐                  │
│  │ Local Cache  │  │  Thumbnail Gen │                  │
│  │ (SSD/NVMe)   │  │  (ffmpeg/vips) │                  │
│  └──────────────┘  └────────────────┘                  │
└──────────────────────────────────────────────────────┘
          │                        │
    ┌─────┴─────┐          ┌──────┴──────┐
    │ S3 Storage │          │ Local Disk  │
    │ (Originals)│          │ (Cache Dir) │
    └───────────┘          └─────────────┘
```

## Success Metrics
- Gallery page load (100 thumbnails): **< 500ms**
- Single photo upload + thumbnail generation: **< 2s**
- Map view with 10,000 geo-tagged photos: **< 1s to render**
- Memory usage at idle: **< 200MB**
- Works on Raspberry Pi 4 (4GB RAM)