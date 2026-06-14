// Shared AIP-158 cursor-pagination vocabulary. The generic useCursorList wrapper
// was removed in favor of calling connect-query's useInfiniteQuery directly per
// hook (exact typing, no casts); these types remain as the shared contract for
// list features and any future pagination helper.
export type {
  CursorPageRequest,
  CursorPageResponse,
  PageExtractor,
} from "./types";
export { DEFAULT_PAGE_SIZE } from "./types";
