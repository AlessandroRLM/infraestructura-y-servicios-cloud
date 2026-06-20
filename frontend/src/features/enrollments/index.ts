// Placeholder — will be rewritten in WU4
export { EnrollmentsPage } from "./components/EnrollmentsPage";

// WU2 foundation exports
export { EnrollmentStatusBadge } from "./components/EnrollmentStatusBadge";
export {
  useEnrollments,
  useOwnEnrollments,
  useCreateEnrollment,
  useMarkEnrollmentPaid,
  useCancelEnrollment,
  mapCreateEnrollmentError,
  mapLifecycleError,
} from "./hooks";
export {
  adminEnrollmentsSearchSchema,
  ownEnrollmentsSearchSchema,
} from "./schemas/search";
export type { AdminEnrollmentsSearch, OwnEnrollmentsSearch } from "./schemas/search";
export { ENROLLMENT_PAGE_SIZE, SEARCH_DEBOUNCE_MS } from "./constants";
