package connectutil_test

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/connectutil"
)

func TestParseUUID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		errCode  connect.Code
		errMsg   string
		wantUUID uuid.UUID
	}{
		{
			name:     "valid lowercase UUID",
			input:    "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			wantUUID: uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
		},
		{
			name:     "valid uppercase UUID is normalized",
			input:    "6BA7B810-9DAD-11D1-80B4-00C04FD430C8",
			wantUUID: uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
		},
		{
			name:     "nil UUID (all zeros) is a valid UUID value",
			input:    "00000000-0000-0000-0000-000000000000",
			wantUUID: uuid.UUID{},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
			errCode: connect.CodeInvalidArgument,
			errMsg:  "invalid UUID",
		},
		{
			name:    "malformed UUID",
			input:   "not-a-uuid",
			wantErr: true,
			errCode: connect.CodeInvalidArgument,
			errMsg:  "invalid UUID",
		},
		{
			name:    "too short UUID",
			input:   "6ba7b810-9dad-11d1-80b4",
			wantErr: true,
			errCode: connect.CodeInvalidArgument,
			errMsg:  "invalid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := connectutil.ParseUUID(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseUUID(%q) = %v, nil; want error", tt.input, got)
				}
				var ce *connect.Error
				if !errors.As(err, &ce) {
					t.Fatalf("ParseUUID(%q) error is not *connect.Error: %T %v", tt.input, err, err)
				}
				if ce.Code() != tt.errCode {
					t.Errorf("ParseUUID(%q) code = %v, want %v", tt.input, ce.Code(), tt.errCode)
				}
				if ce.Message() != tt.errMsg {
					t.Errorf("ParseUUID(%q) message = %q, want %q", tt.input, ce.Message(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseUUID(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.wantUUID {
				t.Errorf("ParseUUID(%q) = %v, want %v", tt.input, got, tt.wantUUID)
			}
		})
	}
}
