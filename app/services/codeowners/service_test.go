package codeowners

import (
	"reflect"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/gitrpc"
)

func TestService_ParseCodeOwner(t *testing.T) {
	content1 := "**/contracts/openapi/v1/ mankrit.singh@harness.io ashish.sanodia@harness.io\n"
	content2 := "**/contracts/openapi/v1/ mankrit.singh@harness.io ashish.sanodia@harness.io\n/scripts/api mankrit.singh@harness.io ashish.sanodia@harness.io"
	content3 := "# codeowner file \n**/contracts/openapi/v1/ mankrit.singh@harness.io ashish.sanodia@harness.io\n#\n/scripts/api mankrit.singh@harness.io ashish.sanodia@harness.io"
	type fields struct {
		repoStore store.RepoStore
		git       gitrpc.Interface
		Config    Config
	}
	type args struct {
		codeOwnersContent string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []codeOwnerDetail
		wantErr bool
	}{
		{
			name: "Code owners Single",
			args: args{codeOwnersContent: content1},
			want: []codeOwnerDetail{{
				Pattern: "**/contracts/openapi/v1/",
				Owners:  []string{"mankrit.singh@harness.io", "ashish.sanodia@harness.io"},
			},
			},
		},
		{
			name: "Code owners Multiple",
			args: args{codeOwnersContent: content2},
			want: []codeOwnerDetail{{
				Pattern: "**/contracts/openapi/v1/",
				Owners:  []string{"mankrit.singh@harness.io", "ashish.sanodia@harness.io"},
			},
				{
					Pattern: "/scripts/api",
					Owners:  []string{"mankrit.singh@harness.io", "ashish.sanodia@harness.io"},
				},
			},
		},
		{
			name: "Code owners With comments",
			args: args{codeOwnersContent: content3},
			want: []codeOwnerDetail{{
				Pattern: "**/contracts/openapi/v1/",
				Owners:  []string{"mankrit.singh@harness.io", "ashish.sanodia@harness.io"},
			},
				{
					Pattern: "/scripts/api",
					Owners:  []string{"mankrit.singh@harness.io", "ashish.sanodia@harness.io"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				repoStore: tt.fields.repoStore,
				git:       tt.fields.git,
				Config:    tt.fields.Config,
			}
			got, err := s.ParseCodeOwner(tt.args.codeOwnersContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCodeOwner() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCodeOwner() got = %v, want %v", got, tt.want)
			}
		})
	}
}
