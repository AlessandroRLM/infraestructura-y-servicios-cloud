// WU4 page + table + filter bar
export { EnrollmentsPage } from "./components/EnrollmentsPage";
export { EnrollmentsTable } from "./components/EnrollmentsTable";
export { EnrollmentsFilterBar } from "./components/EnrollmentsFilterBar";

// WU2 foundation exports
export { EnrollmentStatusBadge } from "./components/EnrollmentStatusBadge";

// WU3 component exports
export { ProgramPicker } from "./components/ProgramPicker";
export { CreateEnrollmentDialog } from "./components/CreateEnrollmentDialog";
export { MarkPaidDialog } from "./components/MarkPaidDialog";
export { CancelEnrollmentDialog } from "./components/CancelEnrollmentDialog";
export { PayOwnEnrollmentDialog } from "./components/PayOwnEnrollmentDialog";
export {
  useEnrollments,
  useOwnEnrollments,
  useCreateEnrollment,
  useMarkEnrollmentPaid,
  useMarkOwnEnrollmentPaid,
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
