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
export function makeStubTransport<Services extends DescService[]>(
  ...handlers: {
    [K in keyof Services]: [Services[K], Partial<ServiceImpl<Services[K]>>];
  }
): Transport {
  return createRouterTransport((router) => {
    // The mapped tuple keeps each handler typed against ITS service at the
    // call site; correlation is erased here, where it no longer matters.
    for (const [service, impl] of handlers as [
      DescService,
      Partial<ServiceImpl<DescService>>,
    ][]) {
      router.service(service, impl);
    }
  });
}
