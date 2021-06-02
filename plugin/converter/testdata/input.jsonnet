local stepName = std.extVar("input.my_step");
local image = std.extVar("input.my_image");
local commands = std.extVar("input.my_command");

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
