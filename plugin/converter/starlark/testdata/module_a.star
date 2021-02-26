load("testdata/module_b.star", "pipeline_b")

def main(ctx):
  return [{
    'kind': 'pipeline',
    'type': 'docker',
    'name': 'default'
  }, pipeline_b(ctx)] 