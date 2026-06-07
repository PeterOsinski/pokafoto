export interface FileItem {
  id: string
  originalName: string
  filename: string
  sizeBytes: number
  mimeType: string
  mediaType: string
  durationSec?: number
  takenAt?: string
  createdAt?: string
  folder_id?: string | null
  isAppManaged?: boolean
  deletedAt?: string
  width?: number
  height?: number
  videoStill?: { url: string }
  thumbnails?: {
    sm?: { url: string; width: number; height: number }
    md?: { url: string; width: number; height: number }
    lg?: { url: string; width: number; height: number }
    xl?: { url: string; width: number; height: number }
    preview?: { url: string; width: number; height: number }
    videoStill?: { url: string; width: number; height: number }
    videoProxy?: { url: string; width: number; height: number }
  }
}

export interface FolderEntry {
  id: string
  name: string
  parent_id: string | null
  user_id?: string
  created_at: string
  updated_at?: string
}

export interface FolderTreeNode {
  folder: FolderEntry
  fileCount: number
  hasShares: boolean
  hasPassword?: boolean
  children: FolderTreeNode[]
}

export interface RootNode {
  children: FolderTreeNode[]
}

export type GalleryItem =
  | { type: 'folder'; id: string; folder: FolderTreeNode }
  | { type: 'file';   id: string; file: FileItem; index: number }

export type SortBy = 'created_at' | 'taken_at' | 'size' | 'name'
export type LayoutMode = 'tiles' | 'list' | 'grouped'
