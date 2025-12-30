module github.com/harness/gitness

go 1.24.11

require (
	cloud.google.com/go/storage v1.43.0
	github.com/Masterminds/semver/v3 v3.3.1
	github.com/Masterminds/squirrel v1.5.4
	github.com/ProtonMail/go-crypto v1.3.0
	github.com/adrg/xdg v0.5.0
	github.com/aws/aws-sdk-go v1.55.2
	github.com/bmatcuk/doublestar/v4 v4.6.1
	github.com/coreos/go-semver v0.3.1
	github.com/dchest/uniuri v1.2.0
	github.com/distribution/distribution/v3 v3.0.0-alpha.1
	github.com/distribution/reference v0.6.0
	github.com/docker/distribution v2.8.2+incompatible
	github.com/docker/docker v27.1.1+incompatible
	github.com/docker/go-connections v0.5.0
	github.com/docker/go-units v0.5.0
	github.com/drone-runners/drone-runner-docker v1.8.4-0.20240815103043-c6c3a3e33ce3
	github.com/drone/drone-go v1.7.1
	github.com/drone/drone-yaml v1.2.3
	github.com/drone/funcmap v0.0.0-20190918184546-d4ef6e88376d
	github.com/drone/go-convert v0.0.0-20240821195621-c6d7be7727ec
	github.com/drone/go-generate v0.0.0-20230920014042-6085ee5c9522
	github.com/drone/go-scm v1.38.9
	github.com/drone/runner-go v1.12.0
	github.com/drone/spec v0.0.0-20230920145636-3827abdce961
	github.com/dustin/go-humanize v1.0.1
	github.com/fatih/color v1.17.0
	github.com/gabriel-vasile/mimetype v1.4.4
	github.com/getkin/kin-openapi v0.131.0
	github.com/gliderlabs/ssh v0.3.7
	github.com/go-chi/chi/v5 v5.2.2
	github.com/go-chi/cors v1.2.1
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-redsync/redsync/v4 v4.13.0
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/golang-migrate/migrate/v4 v4.17.1
	github.com/google/go-cmp v0.7.0
	github.com/google/go-jsonnet v0.20.0
	github.com/google/uuid v1.6.0
	github.com/google/wire v0.6.0
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75
	github.com/gorilla/mux v1.8.1
	github.com/gotidy/ptr v1.4.0
	github.com/guregu/null v4.0.0+incompatible
	github.com/harness/harness-migrate v0.44.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/inhies/go-bytesize v0.0.0-20220417184213-4913239db9cf
	github.com/jackc/pgerrcode v0.0.0-20240316143900-6e2875d9b438
	github.com/jackc/pgx/v5 v5.5.5
	github.com/jmoiron/sqlx v1.4.0
	github.com/joho/godotenv v1.5.1
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/klauspost/compress v1.17.8
	github.com/lib/pq v1.10.9
	github.com/lucasb-eyer/go-colorful v1.2.0
	github.com/maragudk/migrate v0.4.3
	github.com/matoous/go-nanoid v1.5.1
	github.com/matoous/go-nanoid/v2 v2.1.0
	github.com/mattn/go-isatty v0.0.20
	github.com/mattn/go-sqlite3 v1.14.22
	github.com/oapi-codegen/runtime v1.1.1
	github.com/onsi/ginkgo/v2 v2.11.0
	github.com/onsi/gomega v1.27.10
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.0
	github.com/pelletier/go-toml/v2 v2.2.2
	github.com/pkg/errors v0.9.1
	github.com/posthog/posthog-go v1.3.3
	github.com/rs/xid v1.5.0
	github.com/rs/zerolog v1.33.0
	github.com/sassoftware/go-rpmutils v0.4.0
	github.com/sercand/kuberesolver/v5 v5.1.1
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.10.0
	github.com/swaggest/openapi-go v0.2.23
	github.com/swaggest/swgui v1.8.1
	github.com/swaggo/http-swagger v1.3.4
	github.com/swaggo/swag v1.16.2
	github.com/tidwall/jsonc v0.3.2
	github.com/ulikunitz/xz v0.5.12
	github.com/unrolled/secure v1.15.0
	github.com/zricethezav/gitleaks/v8 v8.18.5-0.20240912004812-e93a7c0d2604
	go.starlark.net v0.0.0-20231121155337-90ade8b19d09
	go.uber.org/multierr v1.11.0
	golang.org/x/crypto v0.40.0
	golang.org/x/exp v0.0.0-20250531010427-b6e5de432a8b
	golang.org/x/oauth2 v0.30.0
	golang.org/x/sync v0.16.0
	golang.org/x/term v0.33.0
	golang.org/x/text v0.27.0
	google.golang.org/api v0.189.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/mail.v2 v2.3.1
	oras.land/oras-go/v2 v2.5.0
)

require (
	cloud.google.com/go v0.115.0 // indirect
	cloud.google.com/go/auth v0.7.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.3 // indirect
	cloud.google.com/go/compute/metadata v0.5.0 // indirect
	cloud.google.com/go/iam v1.1.12 // indirect
	dario.cat/mergo v1.0.2 // indirect
	github.com/99designs/httpsignatures-go v0.0.0-20170731043157-88528bf4ca7e // indirect
	github.com/BobuSumisu/aho-corasick v1.0.3 // indirect
	github.com/DataDog/zstd v1.5.5 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/anmitsu/go-shlex v0.0.0-20200514113438-38f4b401e2be // indirect
	github.com/antonmedv/expr v1.15.5 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bmatcuk/doublestar v1.3.4 // indirect
	github.com/buildkite/yaml v2.1.0+incompatible // indirect
	github.com/charmbracelet/lipgloss v0.12.1 // indirect
	github.com/charmbracelet/x/ansi v0.1.4 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/drone/envsubst v1.0.3 // indirect
	github.com/fatih/semgroup v1.2.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/gitleaks/go-gitdiff v0.9.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/spec v0.20.9 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.13.0 // indirect
	github.com/h2non/filetype v1.1.3 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/muesli/termenv v0.15.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/natessilva/dag v0.0.0-20180124060714-7194b8dcc5c4 // indirect
	github.com/oasdiff/yaml v0.0.0-20250309154309-f31be36b4037 // indirect
	github.com/oasdiff/yaml3 v0.0.0-20250309153720-d2182401db90 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/prometheus/client_golang v1.19.1 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sagikazarmark/locafero v0.6.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.19.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/swaggo/files v0.0.0-20220728132757-551d4a08d97a // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.51.0 // indirect
	go.opentelemetry.io/otel v1.28.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.opentelemetry.io/otel/trace v1.28.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240723171418-e6d459c13d2a // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240723171418-e6d459c13d2a // indirect
	google.golang.org/grpc v1.65.0 // indirect
	google.golang.org/protobuf v1.35.2 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

require (
	github.com/go-logr/logr v1.4.2
	github.com/go-logr/zerologr v1.2.3
	github.com/mattn/go-colorable v0.1.13 // indirect
)

require (
	cloud.google.com/go/profiler v0.3.1
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/djherbis/buffer v1.2.0
	github.com/djherbis/nio/v3 v3.0.1
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/pprof v0.0.0-20221103000818-d260c55eee4c // indirect
	github.com/google/subcommands v1.2.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/swaggest/jsonschema-go v0.3.40
	github.com/swaggest/refl v1.1.0 // indirect
	github.com/vearutop/statigz v1.4.0 // indirect
	github.com/yuin/goldmark v1.4.13
	golang.org/x/mod v0.26.0
	golang.org/x/net v0.42.0
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/tools v0.34.0 // indirect
	google.golang.org/genproto v0.0.0-20240722135656-d784300faade // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/harness/gitness/registry => ./registry

// Force containerd to a secure version to fix CVE-2024-25621 (requires >= v1.7.29)
replace github.com/containerd/containerd => github.com/containerd/containerd v1.7.29
