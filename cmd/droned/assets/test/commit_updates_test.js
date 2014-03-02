describe("CommitUpdates", function() {
  describe("constructor", function() {
    it("initializes with a socket", function() {
      var socket = {mock: true};
      var updates = new Drone.CommitUpdates(socket);
      expect(updates.socket).toEqual(socket);
    });

    it("initializes with a string", function() {
      window.WebSocket = function(url) { this.url = url; };
      var updates = new Drone.CommitUpdates('/path');
      expect(updates.socket.url).toEqual('ws://localhost/path');
    });

    it("attaches handlers to the socket", function() {
      var socket = {mock: true};
      var updates = new Drone.CommitUpdates(socket);
      expect(typeof(socket.onmessage)).toEqual("function");
    });
  });

  describe("onMessage", function() {
    it("appends to the lineBuffer", function() {
      var updates = new Drone.CommitUpdates({});
      updates.lineBuffer = "foo ";
      updates.onMessage({data: 'bar'});
      expect(updates.lineBuffer).toEqual('foo bar');
    });
  });

  describe("updateScreen", function() {
    it("does nothing when the lineBuffer is empty", function() {
      var updates = new Drone.CommitUpdates({});
      var el = document.createElement('div');
      expect(el.innerHTML).toEqual('');
      updates.startOutput(el);
      expect(el.innerHTML).toEqual('');
      updates.stopOutput();
    });

    it("writes the lineBuffer to the element", function() {
      var socket = {};
      var updates = new Drone.CommitUpdates(socket);
      socket.onmessage({data: 'foo'});
      socket.onmessage({data: ' bar'});

      var el = document.createElement('div');
      expect(el.innerHTML).toEqual('');

      updates.startOutput(el);
      expect(el.innerHTML).toEqual('foo bar');
      updates.stopOutput();
    });
  });
});
