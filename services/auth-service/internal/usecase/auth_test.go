package usecase

import "testing"

func TestValidateRegistration(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		user    string
		display string
		pass    string
		wantErr bool
	}{
		{
			name:    "valid",
			email:   "user@sahiy.stream",
			user:    "streamer1",
			display: "Streamer",
			pass:    "password123",
			wantErr: false,
		},
		{
			name:    "short password",
			email:   "user@sahiy.stream",
			user:    "streamer1",
			display: "Streamer",
			pass:    "short",
			wantErr: true,
		},
		{
			name:    "invalid email",
			email:   "not-email",
			user:    "streamer1",
			display: "Streamer",
			pass:    "password123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegistration(tt.email, tt.user, tt.display, tt.pass)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateRegistration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
