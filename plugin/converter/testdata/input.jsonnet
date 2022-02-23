local stepName = std.extVar("input.stepName");
local image = std.extVar("input.image");
local commands = std.extVar("input.commands");
local additionalSteps = std.parseYaml(std.extVar("input.additionalSteps"));

{
  "kind": "pipeline",
  "type": "docker",
  "name": "default",
  "steps": [
    {
      "name": stepName,
      "image": image,
      "commands": [
        commands
      ]
    }
  ] + [step for step in additionalSteps]
}
