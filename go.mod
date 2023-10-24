module github.com/harness/gitness

go 1.19

replace github.com/docker/docker => github.com/docker/engine v17.12.0-ce-rc1.0.20200309214505-aa6a9891b09c+incompatible

require (
	cloud.google.com/go/storage v1.33.0
	code.gitea.io/gitea v1.17.2
	github.com/Masterminds/squirrel v1.5.1
	github.com/adrg/xdg v0.3.2
	github.com/aws/aws-sdk-go v1.44.322
	github.com/bmatcuk/doublestar/v4 v4.6.0
	github.com/coreos/go-semver v0.3.0
	github.com/dchest/uniuri v0.0.0-20200228104902-7aecb25e1fe5
	github.com/drone-runners/drone-runner-docker v1.8.4-0.20231017141518-b8f2abebce54
	github.com/drone/drone-go v1.7.1
	github.com/drone/drone-yaml v1.2.3
	github.com/drone/funcmap v0.0.0-20190918184546-d4ef6e88376d
	github.com/drone/go-convert v0.0.0-20230919093251-7104c3bcc635
	github.com/drone/go-generate v0.0.0-20230920014042-6085ee5c9522
	github.com/drone/go-scm v1.31.2
	github.com/drone/runner-go v1.12.0
	github.com/drone/spec v0.0.0-20230919004456-7455b8913ff5
	github.com/go-chi/chi v1.5.4
	github.com/go-chi/cors v1.2.1
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-redsync/redsync/v4 v4.7.1
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/go-cmp v0.5.9
	github.com/google/uuid v1.3.1
	github.com/google/wire v0.5.0
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75
	github.com/gotidy/ptr v1.4.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/guregu/null v4.0.0+incompatible
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jmoiron/sqlx v1.3.3
	github.com/joho/godotenv v1.3.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/lib/pq v1.10.5
	github.com/maragudk/migrate v0.4.1
	github.com/matoous/go-nanoid v1.5.0
	github.com/matoous/go-nanoid/v2 v2.0.0
	github.com/mattn/go-isatty v0.0.17
	github.com/mattn/go-sqlite3 v1.14.12
	github.com/pkg/errors v0.9.1
	github.com/robfig/cron/v3 v3.0.0
	github.com/rs/xid v1.4.0
	github.com/rs/zerolog v1.29.0
	github.com/sercand/kuberesolver/v5 v5.1.0
	github.com/stretchr/testify v1.8.4
	github.com/swaggest/openapi-go v0.2.23
	github.com/swaggest/swgui v1.4.2
	github.com/unrolled/secure v1.0.8
	go.uber.org/multierr v1.8.0
	golang.org/x/crypto v0.13.0
	golang.org/x/exp v0.0.0-20230108222341-4b8118a2686a
	golang.org/x/sync v0.3.0
	golang.org/x/term v0.12.0
	golang.org/x/text v0.13.0
	google.golang.org/api v0.132.0
	google.golang.org/grpc v1.56.2
	google.golang.org/protobuf v1.31.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
)

require (
	cloud.google.com/go v0.110.4 // indirect
	cloud.google.com/go/compute v1.20.1 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v1.1.0 // indirect
	dario.cat/mergo v1.0.0 // indirect
	github.com/99designs/httpsignatures-go v0.0.0-20170731043157-88528bf4ca7e // indirect
	github.com/antonmedv/expr v1.15.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bmatcuk/doublestar v1.3.4 // indirect
	github.com/buildkite/yaml v2.1.0+incompatible // indirect
	github.com/cloudflare/circl v1.3.3 // indirect
	github.com/containerd/containerd v1.7.6 // indirect
	github.com/cyphar/filepath-securejoin v0.2.4 // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker v23.0.3+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/drone/envsubst v1.0.3 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/s2a-go v0.1.4 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.5 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/natessilva/dag v0.0.0-20180124060714-7194b8dcc5c4 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc2.0.20221005185240-3a7f492d3f1b // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/prometheus/client_golang v1.15.1 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/skeema/knownhosts v1.2.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/oauth2 v0.10.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230706204954-ccb25ca9f130 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230711160842-782d3b101e98 // indirect
)

require (
	github.com/go-logr/logr v1.2.4
	github.com/go-logr/zerologr v1.2.3
	github.com/mattn/go-colorable v0.1.13 // indirect
)

require (
	cloud.google.com/go/profiler v0.3.1
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20230828082145-3c4c8a2d2371 // indirect
	github.com/acomagu/bufpipe v1.0.4 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20211218093645-b94a6e3cc137 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/djherbis/buffer v1.2.0 // indirect
	github.com/djherbis/nio/v3 v3.0.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/go-enry/go-enry/v2 v2.8.2 // indirect
	github.com/go-enry/go-oniguruma v1.2.1 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.5.0
	github.com/go-git/go-git/v5 v5.9.0
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golang-jwt/jwt/v4 v4.4.1 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/pprof v0.0.0-20221103000818-d260c55eee4c // indirect
	github.com/google/subcommands v1.2.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-version v1.4.0 // indirect
	github.com/jackc/pgx/v4 v4.12.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/swaggest/jsonschema-go v0.3.40 // indirect
	github.com/swaggest/refl v1.1.0 // indirect
	github.com/vearutop/statigz v1.1.5 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/yuin/goldmark v1.4.13 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/net v0.15.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
	golang.org/x/tools v0.13.0 // indirect
	google.golang.org/genproto v0.0.0-20230706204954-ccb25ca9f130 // indirect
	gopkg.in/ini.v1 v1.66.4 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	strk.kbt.io/projects/go/libravatar v0.0.0-20191008002943-06d1c002b251 // indirect
)
