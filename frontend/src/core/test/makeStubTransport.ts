import type { DescService } from "@bufbuild/protobuf";
import type { ServiceImpl, Transport } from "@connectrpc/connect";
import { createRouterTransport } from "@connectrpc/connect";

/**
 * Creates an in-memory Connect transport for tests.
 * Type-safe against the generated proto contract.
 *
 * Design choice: createRouterTransport (Connect), NOT MSW.
 * Same generated types → tests break at compile time if proto changes.
 *
 * Usage:
 *   const transport = makeStubTransport(
 *     [GradesService, { listOwnGrades: async () => ({ grades: [] }) }],
 *   );
 */
export function makeStubTransport(
  ...handlers: [DescService, Partial<ServiceImpl<DescService>>][]
): Transport {
  return createRouterTransport((router) => {
    for (const [service, impl] of handlers) {
      router.service(service, impl);
    }
  });
}
