import { createConnectQueryKey, useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { IamService } from "@/gen/iam/v1/iam_pb";
import { USER_DETAIL_BASE_KEY } from "../api/queries";

export function useAssignRole(userId: string) {
  const queryClient = useQueryClient();
  return useMutation(IamService.method.assignRole, {
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: [...USER_DETAIL_BASE_KEY, userId, "iam"],
        }),
        queryClient.invalidateQueries({
          // Partial key matches all listUsers infinite queries regardless of input/transport.
          queryKey: createConnectQueryKey({
            schema: IamService.method.listUsers,
            cardinality: "infinite",
          }),
        }),
      ]);
    },
  });
}
