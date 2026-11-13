package mysql

import "testing"

func TestLocaleFormatFunction(t *testing.T) {
	for _, tt := range []struct {
		name      string
		locale    string
		number    string
		precision string
		want      string
		wantErr   bool
	}{
		{name: "US formatting", locale: "en_US", number: "1234567.8", precision: "2", want: "1,234,567.80"},
		{name: "empty precision", locale: "en_US", number: "123.4", precision: "", want: "123"},
		{name: "empty number", locale: "en_US", number: "", precision: "2", want: "0.00"},
		{name: "bare minus", locale: "en_US", number: "-", precision: "2", want: "0.00"},
		{name: "invalid number excessive precision", locale: "en_US", number: "abc", precision: "1000001", wantErr: true},
		{name: "precision allocation limit", locale: "en_US", number: "1", precision: "1000001", wantErr: true},
		{name: "unsupported locale", locale: "zh_CN", number: "123.4", precision: "2", wantErr: true},
		{name: "precision too large", locale: "en_US", number: "123.4", precision: "18446744073709551615", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetLocaleFormatFunction(tt.locale)(tt.number, tt.precision)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %t", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
