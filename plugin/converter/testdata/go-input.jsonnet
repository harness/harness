local goImage = if std.extVar("input.goImage") != "" then std.extVar("input.goImage") else "golang:1.18";
local pipeline_name = if std.extVar("input.pipelineName") != "" then std.extVar("input.pipelineName") else "development";

local pipeline(name) = {
  kind: 'pipeline',
  type: 'docker',
  name: name,
  withTrigger(trigger):: self + {
    trigger: trigger,
  },
  withVolumes(volumes):: self + {
    volumes: volumes,
  },
  withServices(services):: self + {
    services: services,
  },
  withSteps(steps):: self + {
    steps: steps,
  },
};

local step(name, image) = {
  name: name,
  image: image,
  withCommands(commands):: self + {
    commands: commands,
  },
  withWhen(when):: self + {
    when: when,
  },
  withVolumes(volumes):: self + {
    volumes: volumes,
  },
  withEnvs(envs):: self + {
    environment: envs,
  },
  withDeps(deps):: self + {
    depends_on: deps,
  },
  withSettings(settings):: self + {
    settings: settings,
  },
};

local golint = step(
  name='golint',
  image='golangci/golangci-lint:v1.52.0-alpine',
).withCommands(['golangci-lint run']);

local gotest = step(
  name='gotest',
  image=goImage,
).withCommands([
  'go test -v ./...',
]);

local volumes = [
  { name: 'docker', host: { path: '/var/run/docker.sock' } },
];

local development =
  pipeline(pipeline_name)
  .withTrigger({
    branch: ['dev'],
    event: ['pull_request', 'push'],
  })
  .withVolumes(volumes)
  .withSteps([
      golint,
      gotest,
  ]);

[
  development,
]