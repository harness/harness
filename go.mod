module github.com/harness/gitness

go 1.20

replace github.com/docker/docker => github.com/docker/engine v17.12.0-ce-rc1.0.20200309214505-aa6a9891b09c+incompatible

require (
	cloud.google.com/go/storage v1.33.0
	github.com/Masterminds/squirrel v1.5.1
	github.com/adrg/xdg v0.3.2
	github.com/aws/aws-sdk-go v1.44.322
	github.com/bmatcuk/doublestar/v4 v4.6.0
	github.com/coreos/go-semver v0.3.0
	github.com/dchest/uniuri v0.0.0-20200228104902-7aecb25e1fe5
	github.com/drone-runners/drone-runner-docker v1.8.4-0.20240109154718-47375e234554
	github.com/drone/drone-go v1.7.1
	github.com/drone/drone-yaml v1.2.3
	github.com/drone/funcmap v0.0.0-20190918184546-d4ef6e88376d
	github.com/drone/go-convert v0.0.0-20230919093251-7104c3bcc635
	github.com/drone/go-generate v0.0.0-20230920014042-6085ee5c9522
	github.com/drone/go-scm v1.38.0
	github.com/drone/runner-go v1.12.0
	github.com/drone/spec v0.0.0-20230920145636-3827abdce961
	github.com/fatih/color v1.16.0
	github.com/gabriel-vasile/mimetype v1.4.3
	github.com/gliderlabs/ssh v0.3.7
	github.com/go-chi/chi v1.5.4
	github.com/go-chi/cors v1.2.1
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-redsync/redsync/v4 v4.7.1
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/go-cmp v0.5.9
	github.com/google/go-jsonnet v0.20.0
	github.com/google/uuid v1.3.1
	github.com/google/wire v0.5.0
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75
	github.com/gotidy/ptr v1.4.0
	github.com/guregu/null v4.0.0+incompatible
	github.com/harness/harness-migrate v0.21.1-0.20240624210736-65c7e9fbe930
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jmoiron/sqlx v1.3.3
	github.com/joho/godotenv v1.3.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/lib/pq v1.10.5
	github.com/maragudk/migrate v0.4.1
	github.com/matoous/go-nanoid v1.5.0
	github.com/matoous/go-nanoid/v2 v2.0.0
	github.com/mattn/go-isatty v0.0.20
	github.com/mattn/go-sqlite3 v1.14.12
	github.com/pkg/errors v0.9.1
	github.com/rs/xid v1.4.0
	github.com/rs/zerolog v1.29.0
	github.com/sercand/kuberesolver/v5 v5.1.0
	github.com/sirupsen/logrus v1.9.0
	github.com/stretchr/testify v1.8.4
	github.com/swaggest/openapi-go v0.2.23
	github.com/swaggest/swgui v1.8.0
	github.com/unrolled/secure v1.0.8
	github.com/zricethezav/gitleaks/v8 v8.18.5-0.20240614204812-26f34692fac6
	go.starlark.net v0.0.0-20231121155337-90ade8b19d09
	go.uber.org/multierr v1.8.0
	golang.org/x/crypto v0.17.0
	golang.org/x/exp v0.0.0-20230108222341-4b8118a2686a
	golang.org/x/oauth2 v0.10.0
	golang.org/x/sync v0.3.0
	golang.org/x/term v0.15.0
	golang.org/x/text v0.14.0
	google.golang.org/api v0.132.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/mail.v2 v2.3.1
)

require (
	cloud.google.com/go v0.110.4 // indirect
	cloud.google.com/go/compute v1.20.1 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v1.1.0 // indirect
	dario.cat/mergo v1.0.0 // indirect
	github.com/99designs/httpsignatures-go v0.0.0-20170731043157-88528bf4ca7e // indirect
	github.com/BobuSumisu/aho-corasick v1.0.3 // indirect
	github.com/anmitsu/go-shlex v0.0.0-20200514113438-38f4b401e2be // indirect
	github.com/antonmedv/expr v1.15.2 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bmatcuk/doublestar v1.3.4 // indirect
	github.com/buildkite/yaml v2.1.0+incompatible // indirect
	github.com/charmbracelet/lipgloss v0.5.0 // indirect
	github.com/containerd/containerd v1.7.6 // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker v23.0.3+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/drone/envsubst v1.0.3 // indirect
	github.com/fatih/semgroup v1.2.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/gitleaks/go-gitdiff v0.9.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/s2a-go v0.1.4 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.5 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/h2non/filetype v1.1.3 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/jackc/pgx/v4 v4.12.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/muesli/reflow v0.2.1-0.20210115123740-9e1d0d53df68 // indirect
	github.com/muesli/termenv v0.15.1 // indirect
	github.com/natessilva/dag v0.0.0-20180124060714-7194b8dcc5c4 // indirect
	github.com/onsi/gomega v1.27.10 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc2.0.20221005185240-3a7f492d3f1b // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/prometheus/client_golang v1.15.1 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rivo/uniseg v0.4.3 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.8.1 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/time v0.0.0-20220411224347-583f2d630306 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230706204954-ccb25ca9f130 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230711160842-782d3b101e98 // indirect
	google.golang.org/grpc v1.56.2 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

require (
	github.com/go-logr/logr v1.2.4
	github.com/go-logr/zerologr v1.2.3
	github.com/mattn/go-colorable v0.1.13 // indirect
)

require (
	cloud.google.com/go/profiler v0.3.1
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20231202071711-9a357b53e9c9 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/djherbis/buffer v1.2.0
	github.com/djherbis/nio/v3 v3.0.1
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/pprof v0.0.0-20221103000818-d260c55eee4c // indirect
	github.com/google/subcommands v1.2.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/swaggest/jsonschema-go v0.3.40
	github.com/swaggest/refl v1.1.0 // indirect
	github.com/vearutop/statigz v1.4.0 // indirect
	github.com/yuin/goldmark v1.4.13
	go.uber.org/atomic v1.10.0 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/tools v0.13.0 // indirect
	google.golang.org/genproto v0.0.0-20230706204954-ccb25ca9f130 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
)
