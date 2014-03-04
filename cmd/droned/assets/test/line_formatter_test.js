describe("LineFormatter", function() {
  it("passes through output without colors", function() {
    var lineFormatter = new Drone.LineFormatter();
    expect(lineFormatter.format("foo")).toEqual("foo");
  });

  it("sets colors", function() {
    var lineFormatter = new Drone.LineFormatter();
    var input = "\u001B[31;31mthis is red\u001B[0m";
    var expected = '<span style="color:red;">this is red</span>';
    expect(lineFormatter.format(input)).toEqual(expected);
  });

  it("sets multiple colors", function() {
    var lineFormatter = new Drone.LineFormatter();
    var input = "\u001B[31;31mthis is red\u001B[0m\u001B[36mthis is cyan\u001B[0m";
    var expected = '<span style="color:red;">this is red</span><span style="color:cyan;">this is cyan</span>';
    expect(lineFormatter.format(input)).toEqual(expected);
  });

  it("escapes greater than and lesser than symbols", function() {
    var lineFormatter = new Drone.LineFormatter();
    var input = "<blink>hello</blink>";
    var expected = '&lt;blink&gt;hello&lt;/blink&gt;';
    expect(lineFormatter.format(input)).toEqual(expected);

  });
});
