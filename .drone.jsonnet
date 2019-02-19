# defines the target version of Go. Please do not
# update this variable unless it has been previously
# approved on the mailing list.
local golang = "golang:1.11";

# defines a temporary volume so that the Go cache can
# be shared with all pipeine steps.
local volumes = [
    {
        name: "gopath",
        temp: {},
    },
];

# defines the default Go cache location as a volume
# that is mounted into pipeline steps.
local mounts = [
    {
        name: "gopath",
        path: "/go",
    },
];

# defines a pipeline step that builds and publishes
# a docker image to a docker remote registry.
local docker(name, os, arch) = {
    name: "publish_" + name,
    image: "plugins/docker",
    settings: {
        repo: "drone/" + name,
        auto_tag: true,
        auto_tag_suffix: os + "-" + arch,
        username: { from_secret: "docker_username" },
        password: { from_secret: "docker_password" },
        dockerfile: "docker/Dockerfile." + name + "." + os + "." + arch,
    },
    when: {
        event: [ "push", "tag" ],
    },
};

# defines a pipeline step that creates and publishes
# a docker manifest to a docker remote registry.
local manifest(name) = {
    name: name,
    image: "plugins/manifest:1",
    settings: {
        ignore_missing: true,
        spec: "docker/manifest." + name + ".tmpl",
        username: { from_secret: "docker_username" },
        password: { from_secret: "docker_password" },
    },
    when: {
        event: [ "push", "tag" ],
    },
};

# defines a pipeline that builds, tests and publishes
# docker images for the Drone agent, server and controller.
local pipeline(name, os, arch) = {
    kind: "pipeline",
    name: name,
    volumes: volumes,
    platform: {
        os: os,
        arch: arch,
    },
    steps: [
        {
            name: "test",
            image: golang,
            volumes: mounts,
            commands: [ "go test -v ./..." ],
        },
        {
            name: "build",
            image: golang,
            volumes: mounts,
            commands: [
                "go build -o release/"+ os +"/" + arch + "/drone-server",
                "go build -o release/"+ os +"/" + arch + "/drone-agent",
                "go build -o release/"+ os +"/" + arch + "/drone-controller",
            ],
            when: {
                event: [ "push", "tag" ],
            },
        },
        docker("agent", os, arch),
        docker("controller", os, arch),
        docker("server", os, arch),
    ],
};

[
    pipeline("linux-amd64", "linux", "amd64"),
    pipeline("linux-arm", "linux", "arm"),
    pipeline("linux-arm64", "linux", "arm64"),
    {
        kind: "pipeline",
        name: "manifest",
        steps: [
            manifest("server"),
            manifest("agent"),
            manifest("controller"),
        ],
        depends_on: [
          "linux-amd64",
          "linux-arm",
          "linux-arm64",
        ],
    },
]
