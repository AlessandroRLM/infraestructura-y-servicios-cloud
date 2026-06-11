package audit_logs

// Compile-time guard: *postgresRepository must satisfy Repository.
// This test will fail to compile until repository.go defines both types.
var _ Repository = (*postgresRepository)(nil)
