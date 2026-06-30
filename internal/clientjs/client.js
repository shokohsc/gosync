(function() {
  var protocol = location.protocol === "https:" ? "wss:" : "ws:";
  var wsURL = protocol + "//" + location.host + "/__bs/ws";
  var socket = new WebSocket(wsURL);

  socket.onmessage = function(event) {
    var msg = JSON.parse(event.data);
    switch (msg.type) {
      case "reload":
        location.reload();
        break;
      case "css":
        document.querySelectorAll("link[rel=stylesheet]").forEach(function(link) {
          link.href = link.href.split("?")[0] + "?v=" + Date.now();
        });
        break;
      case "scroll":
        window.scrollTo(msg.data.x, msg.data.y);
        break;
    }
  };

  socket.onclose = function() {
    setTimeout(function() {
      location.reload();
    }, 2000);
  };

  var scrollTimer;
  window.addEventListener("scroll", function() {
    clearTimeout(scrollTimer);
    scrollTimer = setTimeout(function() {
      socket.send(JSON.stringify({
        type: "scroll",
        data: { x: window.scrollX, y: window.scrollY }
      }));
    }, 100);
  });
})();
