package admin

//func TestAdmin_parsePunishment(t *testing.T) {
//	tests := []struct {
//		name         string
//		input        string
//		allowInherit bool
//		want         config.Punishment
//		wantErr      bool
//	}{
//		{
//			name:    "delete_symbol",
//			input:   "-",
//			want:    config.Punishment{Action: "delete"},
//			wantErr: false,
//		},
//		{
//			name:         "inherit_allowed",
//			input:        "*",
//			allowInherit: true,
//			want:         config.Punishment{Action: "inherit"},
//			wantErr:      false,
//		},
//		{
//			name:         "inherit_not_allowed",
//			input:        "*",
//			allowInherit: false,
//			wantErr:      true,
//		},
//		{
//			name:    "ban_zero",
//			input:   "0",
//			want:    config.Punishment{Action: "ban"},
//			wantErr: false,
//		},
//		{
//			name:    "timeout_valid",
//			input:   "30",
//			want:    config.Punishment{Action: "timeout", Duration: 30},
//			wantErr: false,
//		},
//		{
//			name:    "timeout_too_high",
//			input:   "1300000",
//			wantErr: true,
//		},
//		{
//			name:    "invalid_string",
//			input:   "abc",
//			wantErr: true,
//		},
//		{
//			name:    "whitespace_trim",
//			input:   " 15 ",
//			want:    config.Punishment{Action: "timeout", Duration: 15},
//			wantErr: false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := parsePunishment(tt.input, tt.allowInherit)
//			if tt.wantErr {
//				assert.Error(t, err)
//			} else {
//				assert.NoError(t, err)
//				assert.Equal(t, tt.want, got)
//			}
//		})
//	}
//}
//
//func TestAdmin_formatPunishment(t *testing.T) {
//	type args struct {
//		punishment config.Punishment
//	}
//	tests := []struct {
//		name string
//		args args
//		want string
//	}{
//		{
//			name: "Delete",
//			args: args{punishment: config.Punishment{Action: "delete"}},
//			want: "удаление сообщения",
//		},
//		{
//			name: "Timeout with duration",
//			args: args{config.Punishment{Action: "timeout", Duration: 60}},
//			want: "таймаут (60)",
//		},
//		{
//			name: "Ban",
//			args: args{config.Punishment{Action: "ban"}},
//			want: "бан",
//		},
//		{
//			name: "Unknown action",
//			args: args{config.Punishment{Action: "something"}},
//			want: "",
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := formatPunishment(tt.args.punishment); got != tt.want {
//				t.Errorf("formatPunishment() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
