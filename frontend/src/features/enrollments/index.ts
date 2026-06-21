// WU4 page + table + filter bar

export { CancelEnrollmentDialog } from "./components/CancelEnrollmentDialog";
export { CreateEnrollmentDialog } from "./components/CreateEnrollmentDialog";
// WU2 foundation exports
export { EnrollmentStatusBadge } from "./components/EnrollmentStatusBadge";
export { EnrollmentsFilterBar } from "./components/EnrollmentsFilterBar";
export { EnrollmentsPage } from "./components/EnrollmentsPage";
export { EnrollmentsTable } from "./components/EnrollmentsTable";
export { MarkPaidDialog } from "./components/MarkPaidDialog";
export { PayOwnEnrollmentDialog } from "./components/PayOwnEnrollmentDialog";
// WU3 component exports
export { ProgramPicker } from "./components/ProgramPicker";
export { SEARCH_DEBOUNCE_MS } from "./constants";
export {
  mapCreateEnrollmentError,
  mapLifecycleError,
  useCancelEnrollment,
  useCreateEnrollment,
  useEnrollments,
  useMarkEnrollmentPaid,
  useMarkOwnEnrollmentPaid,
  useOwnEnrollments,
} from "./hooks";
export type {
  AdminEnrollmentsSearch,
  OwnEnrollmentsSearch,
} from "./schemas/search";
export {
  adminEnrollmentsSearchSchema,
  ownEnrollmentsSearchSchema,
} from "./schemas/search";
