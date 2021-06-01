def main(ctx):
  return {
    "kind": "pipeline",
    "name": "build",
    "steps": [
      {
        "name": ctx.input.stepName,
        "image": ctx.input.image,
        "commands": [
            ctx.input.commands
        ]
      }
    ]
  }
