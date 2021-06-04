local stepName = std.extVar("input.stepName");
local image = std.extVar("input.image");
local commands = std.extVar("input.commands");

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
  ]
}
