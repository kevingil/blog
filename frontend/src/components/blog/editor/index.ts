// Barrel file for the editor module
// Re-exports all extracted editor components and utilities

export { DiffHighlighter } from './diff-highlighter';
export { FormattingToolbar } from './FormattingToolbar';
export { ImageLoader } from './ImageLoader';
export { 
  DEFAULT_IMAGE_PROMPT, 
  articleSchema, 
  getToolDisplayName,
  type ArticleFormData, 
  type ChatMessage, 
  type SearchResult, 
  type SourceInfo 
} from './editor-types';
