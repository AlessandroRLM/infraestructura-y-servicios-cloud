export interface CursorPageRequest {
  pageSize?: number;
  pageToken?: string;
}

export interface CursorPageResponse<TItem> {
  items: TItem[];
  nextPageToken: string;
}

/** Adapts any generated List*Response into the generic cursor shape. */
export type PageExtractor<TRes, TItem> = (
  res: TRes,
) => CursorPageResponse<TItem>;

export const DEFAULT_PAGE_SIZE = 20;
