def main(ctx):
  print(ctx.build)
  print(ctx.build.commit)
  print(ctx.repo)
  print(ctx.repo.namespace)
  print(ctx.repo.name)
  return {
    'kind': 'pipeline',
    'type': 'docker',
    'name': 'default'
  }
