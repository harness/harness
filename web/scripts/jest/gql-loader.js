module.exports = {
  process(src) {
    return (
      'module.exports = ' +
      JSON.stringify(src)
        .replace(/\u2028/g, '\\u2028')
        .replace(/\u2029/g, '\\u2029')
    )
  }
}
