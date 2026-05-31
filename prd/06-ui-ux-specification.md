# 6. UI/UX Specification

## 6.1 Design Principles

1. **Speed-first**: Every interaction must feel instant. Thumbnails appear immediately from cache. No spinners for cached content.
2. **Mobile-native feel**: Touch gestures, bottom navigation on mobile, responsive grid that reflows naturally.
3. **Progressive disclosure**: Show thumbnails first, EXIF on demand, map on demand. Don't overwhelm.
4. **Dark mode default**: Photo apps should use dark backgrounds to make images pop. Light mode as option.
5. **Zero-config look**: Works beautifully out of the box. No setup wizard needed.

---

## 6.2 Layout Structure

### Login / Register Layout
```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│                    ┌─────────────────────┐               │
│                    │   DRIVE Logo        │               │
│                    │                     │               │
│                    │  [Username]         │               │
│                    │  [Password]         │               │
│                    │                     │               │
│                    │  [Log In]           │               │
│                    │                     │               │
│                    │  Don't have an      │               │
│                    │  account? Register  │               │
│                    └─────────────────────┘               │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Desktop Layout (≥1024px, authenticated)
```
┌─────────────────────────────────────────────────────────┐
│  [Drive Logo]  [Gallery] [Folders] [Timeline] [Map] [Admin] [████░░ 1.5GB/10GB]  [👤]  │
├──────────┬──────────────────────────────────────────────┤
│          │                                              │
│  Dir     │          Main Content Area                   │
│  Tree    │                                              │
│          │   ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐       │
│  ├ 2024  │   │    │ │    │ │    │ │    │ │    │       │
│  │ ├ 07  │   └────┘ └────┘ └────┘ └────┘ └────┘       │
│  │ ├ 06  │   ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐       │
│  │ └ 05  │   │    │ │    │ │    │ │    │ │    │       │
│  ├ 2023  │   └────┘ └────┘ └────┘ └────┘ └────┘       │
│  └ files │                                              │
│          │                                              │
├──────────┴──────────────────────────────────────────────┤
│  Status bar: 15,420 photos · 120 GB · Cache 65% full    │
└─────────────────────────────────────────────────────────┘
```

### Mobile Layout (<768px, authenticated)
```
┌─────────────────────┐
│ [Drive]    [🔍] [👤] │  ← Top bar (48px)
├─────────────────────┤
│                     │
│  ┌───┐ ┌───┐ ┌───┐ │
│  │   │ │   │ │   │ │  ← 3-column thumbnail grid
│  └───┘ └───┘ └───┘ │
│  ┌───┐ ┌───┐ ┌───┐ │
│  │   │ │   │ │   │ │
│  └───┘ └───┘ └───┘ │
│  ┌───┐ ┌───┐ ┌───┐ │
│  │   │ │   │ │   │ │
│  └───┘ └───┘ └───┘ │
│                     │
├─────────────────────┤
│ [🏠] [📁] [📅] [🗺]  │  ← Bottom nav (56px)
└─────────────────────┘
```

### Tablet Layout (768px–1023px)
- 4-column thumbnail grid
- Collapsible sidebar (hamburger menu)
- Top navigation tabs

---

## 6.3 View Specifications

### 6.3.0 Login / Register View

**Login Form:**
- Centered card on a dark background with Drive logo
- Username + password fields
- "Log In" primary button
- "Register" link (visible if `auth.allow_registration: true`)
- Error states: "Invalid credentials", "Account locked", "Server unreachable"
- On success: redirect to Gallery view

**Register Form:**
- Username + password + confirm password + display name (optional)
- Validation: username 3-32 chars alphanumeric, password 8+ chars
- "Create Account" button
- "Already have an account? Log In" link
- On success: auto-login and redirect to Gallery view

### 6.3.1 Gallery View

**Thumbnail Grid:**
- CSS Grid with `grid-template-columns: repeat(auto-fill, minmax(200px, 1fr))`
- Thumbnails are square crops (object-fit: cover) on desktop, 3:2 aspect on mobile
- Hover: slight scale (1.02) + shadow + overlay with date and location
- Click: opens Lightbox
- Lazy loading: Intersection Observer with 200px root margin
- Skeleton placeholders while loading (pulsing gray rectangles)

**Inline Upload:**
- An "Upload" button is positioned above the thumbnail grid
- File picker is restricted to image and video MIME types (`image/*,video/*`)
- Uploads from the gallery view go to root (auto-organized by date)
- The dedicated Upload tab has been removed in favor of inline uploads

**Thumbnail Card States:**
| State | Visual |
|---|---|
| Loading | Pulsing skeleton placeholder |
| Loaded | Image with fade-in transition (200ms) |
| Error | Broken image icon with retry button |
| Video | Play icon overlay (▶) + duration badge |
| File | File type icon (PDF, DOC, ZIP, etc.) on colored background |
| Selected | Blue border + checkmark (for batch operations) |

**Sort/Filter Bar:**
```
[📷 Photos] [🎬 Videos] [📄 All Files]  |  [Sort: Date ↓] [🔍 Search...]  |  [☐ All folders]
```
Layout toggle: [Tiles] [List] [Grouped by Day] [Folders]

The **"All folders"** toggle, when enabled, includes files from all user-created folders in the gallery listing (not just root-level files). When disabled (default), only files at the root (no folder assignment) are shown. This filter is essential for "Date Taken" sorting to work across the entire library, allowing users to keep photos in organized folders while still browsing a unified timeline.

**Selection & Batch Operations:**
- Checkboxes appear on thumbnails when navigating the gallery (always visible when any file is selected, hover-only otherwise)
- Click to select/deselect individual files
- Shift+click selects a range between two click positions
- Selected files show a blue accent border with checkmark
- **Action Bar** (sticky, appears when files are selected):
  ```
  [N selected]  [Delete] [Move] [Copy]  [Clear]
  ```
- Delete key triggers batch delete confirmation dialog
- Move/Copy buttons open a **Folder Picker Dialog** — modal with folder tree and inline "New Folder" creation
- Delete confirmation: modal warning about soft-delete (files go to trash, recoverable)

**Folder Tree View (Folders layout):**
- Replaces the thumbnail grid with a folder browser
- Top-level shows folder cards with name, file count, folder icon
- Click a folder card to navigate into it (URL updates: `?layout=folders&folder_id=uuid`)
- "Back" button navigates up to parent
- "+ New Folder" button in the header creates a folder in the current context
- Inline creation: text input + Create/Cancel buttons
- When inside a folder, show its immediate subfolders as cards and files as thumbnails below
- Root-level folders have no parent; nested folders use `parent_id` self-reference
- File counts aggregate recursively (parent shows total including children)

### 6.3.2 Lightbox / Photo Detail

```
┌─────────────────────────────────────────────────────────┐
│  [✕ Close]                          [< Prev] [Next >]  │
│                                                         │
│                                                         │
│                    [Full Preview Image]                 │
│                    (720p, pinch-zoom)                   │
│                                                         │
│                                                         │
├─────────────────────────────────────────────────────────┤
│  [ℹ EXIF] [🗺 Map] [⬇ Download] [🗑 Delete] [↗ Share] │
├─────────────────────────────────────────────────────────┤
│  📷 iPhone 15 Pro · f/1.78 · 1/2500s · ISO 80 · 6.8mm │
│  📍 Warsaw, Poland · 🗓 July 15, 2024 · 14:30         │
└─────────────────────────────────────────────────────────┘
```

**Interactions:**
- **Keyboard**: ← → for prev/next, Esc to close, F for fullscreen
- **Touch**: Swipe left/right for prev/next, pinch to zoom, double-tap to zoom 2x
- **Mouse**: Click sides for prev/next, scroll to zoom
- **EXIF panel**: Slides up from bottom on mobile, side panel on desktop

### 6.3.3 Timeline View

```
┌─────────────────────────────────────────────────────────┐
│                    Timeline                              │
│  ┌──────────────────────────────────────────────────┐   │
│  │  ●── July 2024 (234 photos) ─────────────────    │   │
│  │  │  [🏞] [🏞] [🏞] [🏞] [🏞] [🏞] [🏞] ...     │   │
│  │  ●── June 2024 (189 photos) ────────────────    │   │
│  │  │  [🏞] [🏞] [🏞] [🏞] [🏞] [🏞] ...          │   │
│  │  ●── May 2024 (201 photos) ─────────────────    │   │
│  │  │  [🏞] [🏞] [🏞] [🏞] [🏞] ...               │   │
│  │  ●── April 2024 (156 photos) ───────────────    │   │
│  │     [🏞] [🏞] [🏞] [🏞] ...                     │   │
│  └──────────────────────────────────────────────────┘   │
│                                                         │
│  [Jump to: 📅 Pick a date...]                           │
└─────────────────────────────────────────────────────────┘
```

- Vertical timeline with dots and lines
- Each month shows a horizontal scrollable strip of thumbnails
- Click a month header → opens that month in gallery view
- Date picker for quick jumping to any month/year
- Sticky header with year markers as you scroll

### 6.3.4 Map View

```
┌─────────────────────────────────────────────────────────┐
│  Map View                                    [Heatmap]  │
├─────────────────────────────────────────────────────────┤
│                                                         │
│                    🗺 Full-Screen Map                   │
│                                                         │
│     🔵(12)                    🔵(3)                     │
│                                                         │
│              🔵(156)    🔵(1)  🔵(8)                   │
│                                                         │
│                    🔵(45)                               │
│                                                         │
│  ┌──────────────────────────────────────────────────┐   │
│  │  📍 Warsaw · 156 photos · July 2024              │   │
│  │  [🏞] [🏞] [🏞] [🏞] [🏞] → See all            │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

- **Clusters**: Blue circles with photo count. Size scales with count (min 30px, max 80px).
- **Single markers**: Small thumbnail circles (30px) at high zoom levels.
- **Click cluster**: Zooms in to reveal individual photos.
- **Click marker**: Opens a bottom sheet with photo thumbnail + EXIF summary.
- **Bottom sheet**: Horizontal scrollable strip of photos at that location.
- **Heatmap toggle**: Overlays a density heatmap layer.
- **Timeline scrubber**: Filter map by date range with a dual-handle slider.

### 6.3.5 Upload (Inline Component)

Upload functionality is available via the `InlineUpload` component, used in both Gallery and Folder views. There is no dedicated Upload tab/page.

**Gallery View Upload:**
- "Upload" button above the thumbnail grid
- File picker restricted to `image/*,video/*` MIME types
- Uploads go to root (auto-organized by date); name+size dedup is applied
- Upload progress is tracked via the global upload tracker (bottom-right floating panel)

**Folder View Upload:**
- "Upload" button in the folder header bar
- Accepts all file types (no MIME restriction)
- Files are uploaded directly into the currently browsed folder; no dedup checks applied

**Global Upload Tracker:**
- Floating panel in bottom-right corner showing all active, completed, and failed uploads
- Real-time progress via WebSocket
- Persists across page navigation

### 6.3.6 Admin View (admin role only)

```
┌─────────────────────────────────────────────────────────┐
│  Admin Panel                                             │
│                                                         │
│  ┌──────────────────────────────────────────────────┐   │
│  │  Users                                            │   │
│  │  ┌──────────┬──────────┬────────┬─────────────┐   │   │
│  │  │ Username │ Role     │ Files  │ Size   │ Thumbnails │ Quota      │ Actions     │   │   │
│  │  ├──────────┼──────────┼────────┼─────────────┤   │   │
│  │  │ johndoe  │ Admin    │ 15,420 │ [Edit] [Del]│   │   │
│  │  │ janesmith│ Member   │  3,200 │ [Edit] [Del]│   │   │
│  │  └──────────┴──────────┴────────┴─────────────┘   │   │
│  │                                                    │   │
│  │  Registration: [🟢 Enabled] — Toggle              │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

- Only visible to users with `role: admin`
- User table with role badges (admin=purple, member=blue)
- Delete user: confirmation dialog with warning about file deletion
- Role change: dropdown in edit modal
- Registration toggle: enables/disables the public registration endpoint

**Job History Section:**
```
┌─────────────────────────────────────────────────────────┐
│  Job History                          [Reconcile Thumbs] │
│  [All] [Completed] [Failed] [Skipped] [Processing]      │
│  ┌──────────┬──────────┬────────────┬─────────┬───────┐ │
│  │ Filename │ Status   │ Error/Reas │ Created │ Action│ │
│  ├──────────┼──────────┼────────────┼─────────┼───────┤ │
│  │ DSC01.JPG│ {failed} │ temp_file  │ 17:25   │[Retry]│ │
│  │ IMG_02   │{complete}│ -          │ 17:22   │       │ │
│  │ IMG_03   │{skipped} │ dup_cont   │ 17:20   │       │ │
│  └──────────┴──────────┴────────────┴─────────┴───────┘ │
│  3 of 200 jobs                        [Prev] [Next]     │
└─────────────────────────────────────────────────────────┘
```

- Status badges: completed=green, failed=red, skipped=yellow, processing=blue, queued=gray
- Status filter tabs with counts (e.g., "Failed 34")
- "Retry" button on failed jobs calls `POST /admin/jobs/{id}/retry`
- "Reconcile Thumbnails" button scans for missing thumbnails and creates repair jobs
- Pagination with Prev/Next buttons
- Auto-refreshes every 10 seconds

**Job Status Badge Colors:**
| Status | Color |
|---|---|
| Completed | Green (`bg-green-500/20 text-green-400`) |
| Failed | Red (`bg-red-500/20 text-red-400`) |
| Skipped | Yellow (`bg-yellow-500/20 text-yellow-400`) |
| Processing | Blue (`bg-blue-500/20 text-blue-400`) |
| Queued | Gray (`bg-gray-500/20 text-gray-400`) |

**Worker Pool Section:**
```
┌─────────────────────────────────────────────────────────┐
│  Worker Pool                                            │
│  Workers: 0/2   Queue: 0                                │
│  Completed: 166   Failed: 34   Skipped: 0               │
│  Processing: [job.png 45% - thumbnails]                 │
└─────────────────────────────────────────────────────────┘
```

- Shows live worker pool stats via `GET /admin/workers`
- Processing jobs display filename, progress bar, and current stage
- Auto-refreshes every 5 seconds
- All counters (completed, failed, skipped) are lifetime totals for the current process

---

### 6.3.7 File Viewer (Non-Media Files)

```
┌─────────────────────────────────────────────────────────┐
│  [✕ Close]                          [⬇ Download]        │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  PDF: Browser PDF viewer in iframe                      │
│  JSON: Formatted, syntax-highlighted code block         │
│  Markdown: Rendered HTML content (dark theme)           │
│  CSV: Scrollable data table with sticky header          │
│  TXT: Plain text in monospace pre block                 │
│                                                         │
├─────────────────────────────────────────────────────────┤
│  filename.txt · 45.2 KB · text/plain                    │
└─────────────────────────────────────────────────────────┘
```

**File Type → Viewer Mapping:**
| Extension | MIME Type | Viewer |
|---|---|---|
| `.pdf` | `application/pdf` | PdfViewer — `<iframe>` with blob URL |
| `.md`, `.markdown` | `text/markdown` | MarkdownViewer — rendered via `marked` |
| `.json` | `application/json` | JsonViewer — formatted + syntax highlighted |
| `.csv` | `text/csv` | CsvViewer — parsed as HTML table with sticky header |
| `.txt`, other `text/*` | `text/plain`, etc. | TextViewer — monospace `<pre>` block |

**Interactions:**
- **Keyboard**: Esc to close
- **Download**: Download button in top bar, always available
- **File info bar**: Bottom bar showing file name, size, and MIME type
- **Open from gallery**: Clicking a non-media file thumbnail opens the File Viewer modal

**States:**
| State | Visual |
|---|---|
| Loading | Full-area spinner with "Loading..." text |
| Loaded | Rendered file content |
| Parse error | "Could not render this file. The file may be malformed. [Download raw]" |
| Unsupported type | "No preview available for this file type. [Download]" |
| Download failed | Toast notification "Download failed" |

### Empty States
| View | Empty State |
|---|---|
| Login | N/A (always shown when unauthenticated) |
| Gallery | "No photos yet. Upload your first photo to get started." + inline Upload button (images/videos only) |
| Timeline | "No photos with dates found." |
| Map | "No geo-tagged photos. Photos with GPS data will appear here." |
| Search | "No results for '{query}'. Try a different search term." |

### Error States
| Error | Visual |
|---|---|
| Thumbnail load failed | Gray placeholder with broken image icon + "Retry" button |
| Upload failed | Red progress bar + error message + "Retry" / "Skip" buttons |
| S3 unreachable | Banner at top: "Storage is temporarily unavailable. Cached content still available." |
| File too large | Inline error: "File exceeds 10GB limit" |

### Loading States
- **Initial page load**: Skeleton grid (pulsing gray rectangles matching thumbnail aspect ratio)
- **Infinite scroll**: Small spinner at bottom of grid
- **Map tiles**: Gray tiles that fade in as they load
- **Upload**: See upload view above

---

## 6.5 Responsive Breakpoints

| Breakpoint | Width | Columns | Sidebar | Navigation |
|---|---|---|---|---|
| Mobile S | <375px | 2 | Hidden | Bottom tabs (Home, Folders, Timeline, Map) |
| Mobile L | 375–767px | 3 | Hidden | Bottom tabs (Home, Folders, Timeline, Map) |
| Tablet | 768–1023px | 4 | Collapsible | Top tabs (Gallery, Folders, Timeline, Map) |
| Desktop | 1024–1439px | 5-6 | Visible (250px) | Top tabs (Gallery, Folders, Timeline, Map) |
| Wide | ≥1440px | 6-8 | Visible (300px) | Top tabs (Gallery, Folders, Timeline, Map) |

---

## 6.6 Color System

```
Dark Theme (Default):
  Background:        #0d0d0d
  Surface:           #1a1a1a
  Surface Elevated:  #262626
  Border:            #333333
  Text Primary:      #f5f5f5
  Text Secondary:    #a3a3a3
  Accent:            #3b82f6 (blue)
  Accent Secondary:  #8b5cf6 (purple)
  Success:           #22c55e
  Warning:           #f59e0b
  Error:             #ef4444

Light Theme (Optional):
  Background:        #fafafa
  Surface:           #ffffff
  Surface Elevated:  #f5f5f5
  Border:            #e5e5e5
  Text Primary:      #171717
  Text Secondary:    #737373
  (Accent colors same)
```

---

## 6.7 Typography

- **Font**: Inter (system font stack as fallback: `-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif`)
- **Scale**:
  - `xs`: 12px — badges, metadata
  - `sm`: 14px — secondary text, file sizes
  - `base`: 16px — body text
  - `lg`: 18px — section headers
  - `xl`: 24px — view titles
  - `2xl`: 32px — hero text

---

## 6.8 Micro-interactions

| Interaction | Animation |
|---|---|
| Thumbnail appear | `opacity: 0 → 1` with `transform: scale(0.95) → scale(1)`, 200ms ease-out |
| Thumbnail hover | `transform: scale(1.03)`, `box-shadow` increase, 150ms ease-out |
| Lightbox open | Image scales from thumbnail position to center, 300ms ease-in-out |
| Lightbox close | Reverse of open, 200ms |
| Swipe next/prev | TranslateX with spring physics |
| Upload drop zone | Border pulses blue on dragover |
| Delete | Item shrinks and fades out, 300ms |
| Cluster click | Map zoom with smooth flyTo, 500ms |
| Bottom sheet | Slides up from bottom, 250ms ease-out |
| Toast notification | Slides in from top-right, auto-dismiss 3s |