def main(ctx):
  return {
    "kind": "pipeline",
    "name": "default",
    "steps": [
      ctx.input.builds
    ]
  }