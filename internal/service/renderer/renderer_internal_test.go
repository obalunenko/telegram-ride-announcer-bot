package renderer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTemplate(t *testing.T) {
	type args struct {
		name     string
		template string
	}

	type expected struct {
		wantNil bool
		wantErr assert.ErrorAssertionFunc
	}
	testCases := []struct {
		name string
		args args
		want expected
	}{
		{
			name: "Valid template file",
			args: args{
				name:     "help",
				template: "templates/help.gotmpl",
			},
			want: expected{
				wantNil: false,
				wantErr: assert.NoError,
			},
		},
		{
			name: "Invalid template file",
			args: args{
				name:     "404",
				template: "404.html",
			},
			want: expected{
				wantNil: true,
				wantErr: assert.Error,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := parseTemplate(tc.args.name, tc.args.template)
			if !tc.want.wantErr(t, err) {
				return
			}

			if tc.want.wantNil {
				assert.Nil(t, tmpl)
			} else {
				assert.NotNil(t, tmpl)
			}
		})
	}
}
