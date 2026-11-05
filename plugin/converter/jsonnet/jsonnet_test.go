package jsonnet

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/drone/drone/core"
)

type fileServiceFunc func(context.Context, *core.User, string, string, string, string) (*core.File, error)

func (f fileServiceFunc) Find(ctx context.Context, user *core.User, repo, commit, ref, path string) (*core.File, error) {
	return f(ctx, user, repo, commit, ref, path)
}

func TestParseRejectsServerFileImports(t *testing.T) {
	file := filepath.Join(t.TempDir(), "server-secret.json")
	if err := os.WriteFile(file, []byte(`{"secret":"server-file-canary"}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, syntax := range []string{"import", "importstr", "importbin"} {
		for _, useTemplate := range []bool{false, true} {
			for _, mode := range []struct {
				service bool
				limit   int
			}{{false, 0}, {false, 10}, {true, 0}, {true, 10}} {
				t.Run(fmt.Sprintf("%s/template=%t/service=%t/limit=%d", syntax, useTemplate, mode.service, mode.limit), func(t *testing.T) {
					source := fmt.Sprintf("{ secret: %s %q }", syntax, filepath.ToSlash(file))
					req := &core.ConvertArgs{
						Repo:   &core.Repository{Config: ".drone.jsonnet"},
						Build:  &core.Build{},
						Config: &core.Config{Data: source},
					}
					var template *core.Template
					if useTemplate {
						template = &core.Template{Name: "pipeline.jsonnet", Data: source}
					}
					var files core.FileService
					if mode.service {
						files = fileServiceFunc(func(context.Context, *core.User, string, string, string, string) (*core.File, error) {
							t.Fatal("absolute server-file import reached the SCM service")
							return nil, nil
						})
					}
					if output, err := Parse(req, files, mode.limit, template, nil); err == nil {
						t.Fatalf("server file import succeeded: %s", output)
					}
				})
			}
		}
	}
}

func TestImporterRejectsPathsOutsideRepository(t *testing.T) {
	for _, tt := range []struct{ from, path string }{
		{".drone.jsonnet", "../secret.jsonnet"},
		{".ci/pipeline.jsonnet", "../../secret.jsonnet"},
		{".drone.jsonnet", "a/../../secret.jsonnet"},
		{".ci/pipeline.jsonnet", "a/../../../secret.jsonnet"},
		{"../pipeline.jsonnet", "secret.jsonnet"},
		{".drone.jsonnet", ".."},
		{".ci/pipeline.jsonnet", "/etc/passwd"},
		{".drone.jsonnet", `..\secret.jsonnet`},
	} {
		t.Run(tt.from+"/"+tt.path, func(t *testing.T) {
			i := &importer{
				repo: &core.Repository{}, build: &core.Build{}, limit: 1,
				fileService: fileServiceFunc(func(context.Context, *core.User, string, string, string, string) (*core.File, error) {
					t.Fatal("invalid import reached the SCM service")
					return nil, nil
				}),
			}
			if _, _, err := i.Import(tt.from, tt.path); err == nil {
				t.Fatal("expected an import path error")
			}
		})
	}
}

func TestParseRepositoryFilenames(t *testing.T) {
	for _, filename := range []string{"file#name%.libsonnet", "file?ref=other.libsonnet", "%2e%2e/secret.libsonnet"} {
		t.Run(filename, func(t *testing.T) {
			files := fileServiceFunc(func(_ context.Context, _ *core.User, repo, commit, ref, path string) (*core.File, error) {
				if repo != "owner/repo" || commit != "commit" || path != filename {
					t.Fatalf("import changed repository coordinates: %q %q %q", repo, commit, path)
				}
				return &core.File{Data: []byte(`{ image: 'alpine' }`)}, nil
			})
			req := &core.ConvertArgs{
				Repo:  &core.Repository{Slug: "owner/repo", Config: ".drone.jsonnet"},
				Build: &core.Build{After: "commit"}, Config: &core.Config{Data: fmt.Sprintf("import %q", filename)},
			}
			if _, err := Parse(req, files, 1, nil, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestParseRepositoryImportsStaySupported(t *testing.T) {
	calls := 0
	files := fileServiceFunc(func(_ context.Context, _ *core.User, repo, commit, ref, path string) (*core.File, error) {
		calls++
		if repo != "owner/repo" || commit != "commit" {
			t.Fatalf("unexpected SCM request: %q %q %q", repo, commit, path)
		}
		switch path {
		case ".ci/pipeline.libsonnet":
			return &core.File{Data: []byte(`import '../shared.libsonnet'`)}, nil
		case "shared.libsonnet":
			return &core.File{Data: []byte(`{"image":"alpine"}`)}, nil
		default:
			t.Fatalf("unexpected import path: %q", path)
			return nil, nil
		}
	})
	req := &core.ConvertArgs{
		Repo:   &core.Repository{Slug: "owner/repo", Config: ".drone.jsonnet"},
		Build:  &core.Build{After: "commit"},
		Config: &core.Config{Data: `{ first: import '.ci/pipeline.libsonnet', second: import '.ci/pipeline.libsonnet' }`},
	}
	if _, err := Parse(req, files, 2, nil, nil); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("want two cached SCM requests, got %d", calls)
	}
}

func TestParse(t *testing.T) {
	before, err := ioutil.ReadFile("../testdata/input.jsonnet")
	if err != nil {
		t.Error(err)
		return
	}

	after, err := ioutil.ReadFile("../testdata/input.jsonnet.golden")
	if err != nil {
		t.Error(err)
		return
	}

	req := &core.ConvertArgs{
		Build: &core.Build{
			After: "3d21ec53a331a6f037a91c368710b99387d012c1",
		},
		Repo: &core.Repository{
			Slug:   "octocat/hello-world",
			Config: ".drone.yml",
		},
		Config: &core.Config{},
	}

	template := &core.Template{
		Name: "my_template.jsonnet",
		Data: string(before),
	}

	templateData := map[string]interface{}{
		"stepName": "my_step",
		"image":    "my_image",
		"commands": "my_command",
	}

	req.Config.Data = string(before)

	got, err := Parse(req, nil, 0, template, templateData)
	if err != nil {
		t.Error(err)
		return
	}

	want := string(after)
	// on windows line endings are \r\n, lets change them to linux for comparison
	if runtime.GOOS == "windows" {
		want = strings.Replace(want, "\r\n", "\n", -1)
	}

	if want != got {
		t.Errorf("Want %q got %q", want, got)
	}
}

func TestParseJsonnetNotTemplateFile(t *testing.T) {
	before, err := ioutil.ReadFile("../testdata/single.jsonnet")
	if err != nil {
		t.Error(err)
		return
	}

	after, err := ioutil.ReadFile("../testdata/input.jsonnet.golden")
	if err != nil {
		t.Error(err)
		return
	}

	req := &core.ConvertArgs{
		Build: &core.Build{
			After: "3d21ec53a331a6f037a91c368710b99387d012c1",
		},
		Repo: &core.Repository{
			Slug:   "octocat/hello-world",
			Config: ".drone.jsonnet",
		},
		Config: &core.Config{},
	}

	req.Repo.Config = "plugin.jsonnet"
	req.Config.Data = string(before)

	got, err := Parse(req, nil, 0, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}

	want := string(after)
	// on windows line endings are \r\n, lets change them to linux for comparison
	if runtime.GOOS == "windows" {
		want = strings.Replace(want, "\r\n", "\n", -1)
	}

	if want != got {
		t.Errorf("Want %q got %q", want, got)
	}
}
