import { Code, ConnectError } from "@connectrpc/connect";
import { useTransport } from "@connectrpc/connect-query";
import { useQuery } from "@tanstack/react-query";
import { ownProfileQueryOptions } from "../api/queries";
import { createRpcProfileSource } from "../api/rpc";

export function useOwnProfile() {
  const transport = useTransport();
  const source = createRpcProfileSource(transport);
  const result = useQuery(ownProfileQueryOptions(source));

  const isNotFound =
    result.isError &&
    result.error instanceof ConnectError &&
    result.error.code === Code.NotFound;

  return {
    profile: result.data,
    isLoading: result.isLoading,
    isError: result.isError,
    isNotFound,
    refetch: result.refetch,
  };
}
