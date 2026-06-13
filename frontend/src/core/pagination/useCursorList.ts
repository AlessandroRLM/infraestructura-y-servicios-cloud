import type {
  DescMessage,
  DescMethodUnary,
  MessageInitShape,
  MessageShape,
} from "@bufbuild/protobuf";
import { useInfiniteQuery } from "@connectrpc/connect-query";
import type { InfiniteData } from "@tanstack/react-query";
import { DEFAULT_PAGE_SIZE, type PageExtractor } from "./types";

/**
 * Thin wrapper over connect-query useInfiniteQuery for AIP-158 cursor pagination.
 * The input message MUST have pageToken (string) and pageSize (number) fields.
 *
 * pageParamKey="pageToken" drives the connect-query infinite query mechanism;
 * getNextPageParam returns `undefined` when nextPageToken is empty, stopping pagination.
 */
export function useCursorList<
  I extends DescMessage,
  O extends DescMessage,
  TItem,
>(
  methodDescriptor: DescMethodUnary<I, O>,
  {
    extract,
    pageSize = DEFAULT_PAGE_SIZE,
    baseInput = {},
  }: {
    extract: PageExtractor<MessageShape<O>, TItem>;
    pageSize?: number;
    baseInput?: Record<string, unknown>;
    [key: string]: unknown;
  },
) {
  // Cast to satisfy connect-query's strict input typing.
  // The caller is responsible for using this with a method whose input has pageToken/pageSize.
  const input = {
    ...baseInput,
    pageSize,
    pageToken: "",
  } as unknown as MessageInitShape<I> &
    Required<
      Pick<
        MessageInitShape<I>,
        "pageToken" extends keyof MessageInitShape<I> ? "pageToken" : never
      >
    >;

  const result = useInfiniteQuery(
    methodDescriptor,
    // biome-ignore lint/suspicious/noExplicitAny: connect-query generic constraint requires the typed PageParam key; runtime is correct
    input as any,
    {
      // biome-ignore lint/suspicious/noExplicitAny: "pageToken" is the AIP-158 page param key; connect-query's generic requires a narrower keyof type
      pageParamKey: "pageToken" as any,
      getNextPageParam: (lastPage: MessageShape<O>) =>
        extract(lastPage).nextPageToken || undefined,
    },
  );

  const allPages =
    (result.data as InfiniteData<MessageShape<O>> | undefined)?.pages ?? [];
  const items: TItem[] = allPages.flatMap((page) => extract(page).items);

  return {
    ...result,
    items,
  };
}
