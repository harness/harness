module github.com/harness/gitness

go 1.20

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
	github.com/drone-runners/drone-runner-docker v1.8.4-0.20240109154718-47375e234554
	github.com/drone/drone-go v1.7.1
	github.com/drone/drone-yaml v1.2.3
	github.com/drone/funcmap v0.0.0-20190918184546-d4ef6e88376d
	github.com/drone/go-convert v0.0.0-20230919093251-7104c3bcc635
	github.com/drone/go-generate v0.0.0-20230920014042-6085ee5c9522
	github.com/drone/go-scm v1.31.2
	github.com/drone/runner-go v1.12.0
	github.com/drone/spec v0.0.0-20230919004456-7455b8913ff5
	github.com/gabriel-vasile/mimetype v1.4.3
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
	github.com/rs/xid v1.4.0
	github.com/rs/zerolog v1.29.0
	github.com/sercand/kuberesolver/v5 v5.1.0
	github.com/sirupsen/logrus v1.9.0
	github.com/stretchr/testify v1.8.4
	github.com/swaggest/openapi-go v0.2.23
	github.com/swaggest/swgui v1.8.0
	github.com/unrolled/secure v1.0.8
	go.starlark.net v0.0.0-20231121155337-90ade8b19d09
	go.uber.org/multierr v1.8.0
	golang.org/x/crypto v0.14.0
	golang.org/x/exp v0.0.0-20230108222341-4b8118a2686a
	golang.org/x/oauth2 v0.10.0
	golang.org/x/sync v0.3.0
	golang.org/x/term v0.13.0
	golang.org/x/text v0.13.0
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
	gitea.com/go-chi/binding v0.0.0-20220309004920-114340dabecb // indirect
	gitea.com/go-chi/cache v0.2.0 // indirect
	gitea.com/lunny/levelqueue v0.4.2-0.20220729054728-f020868cc2f7 // indirect
	github.com/42wim/sshsig v0.0.0-20211121163825-841cf5bbc121 // indirect
	github.com/99designs/httpsignatures-go v0.0.0-20170731043157-88528bf4ca7e // indirect
	github.com/RoaringBitmap/roaring v0.9.4 // indirect
	github.com/alecthomas/chroma v0.10.0 // indirect
	github.com/antonmedv/expr v1.15.2 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bgentry/speakeasy v0.1.0 // indirect
	github.com/bits-and-blooms/bitset v1.2.2 // indirect
	github.com/blevesearch/bleve/v2 v2.3.2 // indirect
	github.com/blevesearch/bleve_index_api v1.0.1 // indirect
	github.com/blevesearch/go-porterstemmer v1.0.3 // indirect
	github.com/blevesearch/gtreap v0.1.1 // indirect
	github.com/blevesearch/mmap-go v1.0.3 // indirect
	github.com/blevesearch/scorch_segment_api/v2 v2.1.0 // indirect
	github.com/blevesearch/segment v0.9.0 // indirect
	github.com/blevesearch/snowballstem v0.9.0 // indirect
	github.com/blevesearch/upsidedown_store_api v1.0.1 // indirect
	github.com/blevesearch/vellum v1.0.7 // indirect
	github.com/blevesearch/zapx/v11 v11.3.3 // indirect
	github.com/blevesearch/zapx/v12 v12.3.3 // indirect
	github.com/blevesearch/zapx/v13 v13.3.3 // indirect
	github.com/blevesearch/zapx/v14 v14.3.3 // indirect
	github.com/blevesearch/zapx/v15 v15.3.3 // indirect
	github.com/bmatcuk/doublestar v1.3.4 // indirect
	github.com/boombuler/barcode v1.0.1 // indirect
	github.com/bradfitz/gomemcache v0.0.0-20190913173617-a41fca850d0b // indirect
	github.com/buildkite/yaml v2.1.0+incompatible // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cloudflare/cfssl v1.6.1 // indirect
	github.com/cloudflare/circl v1.3.3 // indirect
	github.com/cncf/udpa/go v0.0.0-20220112060539-c52dc94e7fbe // indirect
	github.com/cncf/xds/go v0.0.0-20230607035331-e9ce68804cb4 // indirect
	github.com/containerd/containerd v1.7.6 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/cyphar/filepath-securejoin v0.2.4 // indirect
	github.com/denisenkom/go-mssqldb v0.12.0 // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker v23.0.3+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/drone/envsubst v1.0.3 // indirect
	github.com/duo-labs/webauthn v0.0.0-20220330035159-03696f3d4499 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/editorconfig/editorconfig-core-go/v2 v2.4.4 // indirect
	github.com/envoyproxy/go-control-plane v0.11.1-0.20230524094728-9239064ad72f // indirect
	github.com/envoyproxy/protoc-gen-validate v0.10.1 // indirect
	github.com/ethantkoenig/rupture v1.0.1 // indirect
	github.com/form3tech-oss/jwt-go v3.2.3+incompatible // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/fullstorydev/grpcurl v1.8.1 // indirect
	github.com/fxamacker/cbor/v2 v2.4.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/goccy/go-json v0.9.7 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/gogs/chardet v0.0.0-20211120154057-b7413eaefb8f // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.0.0-20170517235910-f1bb20e5a188 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/certificate-transparency-go v1.1.2-0.20210511102531-373a877eec92 // indirect
	github.com/google/s2a-go v0.1.4 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.5 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/gorilla/css v1.0.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jaytaylor/html2text v0.0.0-20211105163654-bc68cce691ba // indirect
	github.com/jhump/protoreflect v1.8.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4 // indirect
	github.com/klauspost/compress v1.16.0 // indirect
	github.com/klauspost/cpuid/v2 v2.0.12 // indirect
	github.com/lunny/dingtalk_webhook v0.0.0-20171025031554-e3534c89ef96 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/microcosm-cc/bluemonday v1.0.19 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/minio-go/v7 v7.0.26 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/natessilva/dag v0.0.0-20180124060714-7194b8dcc5c4 // indirect
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/oliamb/cutter v0.2.2 // indirect
	github.com/olivere/elastic/v7 v7.0.32 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc2.0.20221005185240-3a7f492d3f1b // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/pquerna/otp v1.3.0 // indirect
	github.com/prometheus/client_golang v1.15.1 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.0.0 // indirect
	github.com/skeema/knownhosts v1.2.0 // indirect
	github.com/soheilhy/cmux v0.1.5 // indirect
	github.com/spf13/cobra v1.3.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/ssor/bom v0.0.0-20170718123548-6386211fdfcf // indirect
	github.com/syndtr/goleveldb v1.0.0 // indirect
	github.com/tmc/grpc-websocket-proxy v0.0.0-20201229170055-e5319fda7802 // indirect
	github.com/unknwon/com v1.0.1 // indirect
	github.com/unrolled/render v1.4.1 // indirect
	github.com/urfave/cli v1.22.12 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	github.com/yohcop/openid-go v1.0.0 // indirect
	github.com/yuin/goldmark-highlighting v0.0.0-20220208100518-594be1970594 // indirect
	github.com/yuin/goldmark-meta v1.1.0 // indirect
	go.etcd.io/bbolt v1.3.7 // indirect
	go.etcd.io/etcd/api/v3 v3.5.1 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.1 // indirect
	go.etcd.io/etcd/client/v2 v2.305.1 // indirect
	go.etcd.io/etcd/client/v3 v3.5.0-alpha.0 // indirect
	go.etcd.io/etcd/etcdctl/v3 v3.5.0-alpha.0 // indirect
	go.etcd.io/etcd/pkg/v3 v3.5.0-alpha.0 // indirect
	go.etcd.io/etcd/raft/v3 v3.5.0-alpha.0 // indirect
	go.etcd.io/etcd/server/v3 v3.5.0-alpha.0 // indirect
	go.etcd.io/etcd/tests/v3 v3.5.0-alpha.0 // indirect
	go.etcd.io/etcd/v3 v3.5.0-alpha.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.uber.org/zap v1.21.0 // indirect
	golang.org/x/time v0.0.0-20220411224347-583f2d630306 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230706204954-ccb25ca9f130 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230711160842-782d3b101e98 // indirect
	google.golang.org/grpc v1.56.2 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/cheggaaa/pb.v1 v1.0.28 // indirect
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df // indirect
	mvdan.cc/xurls/v2 v2.4.0 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
	xorm.io/builder v0.3.11 // indirect
	xorm.io/xorm v1.3.2-0.20220714055524-c3bce556200f // indirect
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
	github.com/djherbis/buffer v1.2.0
	github.com/djherbis/nio/v3 v3.0.1
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/go-enry/go-enry/v2 v2.8.2 // indirect
	github.com/go-enry/go-oniguruma v1.2.1 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.5.0 // indirect
	github.com/go-git/go-git/v5 v5.9.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golang-jwt/jwt/v4 v4.4.1 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/pprof v0.0.0-20221103000818-d260c55eee4c // indirect
	github.com/google/subcommands v1.2.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-version v1.4.0 // indirect
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
	github.com/vearutop/statigz v1.4.0 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/yuin/goldmark v1.4.13 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/tools v0.13.0 // indirect
	google.golang.org/genproto v0.0.0-20230706204954-ccb25ca9f130 // indirect
	gopkg.in/ini.v1 v1.66.4 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	strk.kbt.io/projects/go/libravatar v0.0.0-20191008002943-06d1c002b251 // indirect
)
