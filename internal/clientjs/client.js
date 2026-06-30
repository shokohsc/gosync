(function() {
  'use strict';

  var protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
  var wsURL = protocol + '//' + location.host + '/__bs/ws';
  var socket = null;
  var reconnectAttempts = 0;
  var maxReconnectAttempts = 50;
  var reconnectDelay = 1000;
  var options = {};

  function connect() {
    socket = new WebSocket(wsURL);

    socket.onopen = function() {
      reconnectAttempts = 0;
    };

    socket.onmessage = function(event) {
      var msg = JSON.parse(event.data);
      handleMessage(msg);
    };

    socket.onclose = function() {
      scheduleReconnect();
    };

    socket.onerror = function() {
      socket.close();
    };
  }

  function scheduleReconnect() {
    if (reconnectAttempts >= maxReconnectAttempts) return;
    reconnectAttempts++;
    var delay = Math.min(reconnectDelay * Math.pow(2, reconnectAttempts), 30000);
    setTimeout(connect, delay);
  }

  function handleMessage(msg) {
    switch (msg.type) {
      case 'hello':
        options = msg.data || {};
        break;
      case 'reload':
      case 'browser:reload':
        location.reload();
        break;
      case 'browser:location':
        location.href = msg.data.url;
        break;
      case 'browser:notify':
        showNotification(msg.data.message || '', msg.data.timeout || 5000);
        break;
      case 'css':
        document.querySelectorAll('link[rel=stylesheet]').forEach(function(link) {
          link.href = link.href.split('?')[0] + '?v=' + Date.now();
        });
        break;
      case 'scroll':
        if (msg.data) window.scrollTo(msg.data.x || 0, msg.data.y || 0);
        break;
      case 'click':
        simulateClick(msg.data);
        break;
      case 'input:text':
        setInputValue(msg.data);
        break;
      case 'input:toggles':
        setToggleValue(msg.data);
        break;
      case 'form:submit':
        dispatchFormEvent(msg.data, 'submit');
        break;
      case 'form:reset':
        dispatchFormReset(msg.data);
        break;
    }
  }

  function send(event) {
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify(event));
    }
  }

  var scrollTimer;
  window.addEventListener('scroll', function() {
    clearTimeout(scrollTimer);
    scrollTimer = setTimeout(function() {
      send({
        type: 'scroll',
        data: { x: window.scrollX, y: window.scrollY }
      });
    }, 100);
  });

  document.addEventListener('click', function(e) {
    var target = e.target;
    var tagName = target.tagName ? target.tagName.toLowerCase() : '';
    if (tagName === 'label') {
      if (target.getAttribute('for') || target.querySelector('input, select, textarea')) return;
    }
    if (tagName === 'input' || tagName === 'a') return;
    var index = getElementIndex(target);
    send({
      type: 'click',
      data: { tagName: tagName, index: index }
    });
  }, true);

  document.addEventListener('input', function(e) {
    var target = e.target;
    if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') {
      var type = (target.getAttribute('type') || 'text').toLowerCase();
      if (type === 'text' || type === 'textarea' || type === 'email' || type === 'search' || type === 'url' || type === 'tel' || type === 'password') {
        send({
          type: 'input:text',
          data: { tagName: target.tagName.toLowerCase(), index: getElementIndex(target), value: target.value }
        });
      }
    }
  }, true);

  document.addEventListener('change', function(e) {
    var target = e.target;
    if (target.tagName === 'SELECT') {
      send({
        type: 'input:toggles',
        data: { tagName: 'select', index: getElementIndex(target), value: target.value }
      });
    } else if (target.tagName === 'INPUT' && (target.type === 'checkbox' || target.type === 'radio')) {
      send({
        type: 'input:toggles',
        data: { tagName: 'input', type: target.type, index: getElementIndex(target), checked: target.checked, value: target.value }
      });
    }
  }, true);

  document.addEventListener('submit', function(e) {
    var target = e.target;
    if (target.tagName === 'FORM') {
      send({
        type: 'form:submit',
        data: { tagName: 'form', index: getElementIndex(target) }
      });
    }
  }, true);

  document.addEventListener('reset', function(e) {
    var target = e.target;
    if (target.tagName === 'FORM') {
      send({
        type: 'form:reset',
        data: { tagName: 'form', index: getElementIndex(target) }
      });
    }
  }, true);

  function getElementIndex(el) {
    var parent = el.parentNode;
    if (!parent) return 0;
    var children = parent.children;
    var idx = 0;
    for (var i = 0; i < children.length; i++) {
      if (children[i] === el) return idx;
      if (children[i].tagName === el.tagName) idx++;
    }
    return 0;
  }

  function simulateClick(data) {
    if (!data) return;
    var elements = document.querySelectorAll(data.tagName || '');
    if (elements.length > (data.index || 0)) {
      elements[data.index || 0].click();
    }
  }

  function setInputValue(data) {
    if (!data) return;
    var elements = document.querySelectorAll(data.tagName || 'input');
    if (elements.length <= (data.index || 0)) return;
    var el = elements[data.index || 0];
    var nativeSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value');
    if (nativeSetter) {
      nativeSetter.set.call(el, data.value);
    } else {
      el.value = data.value;
    }
    el.dispatchEvent(new Event('input', { bubbles: true }));
  }

  function setToggleValue(data) {
    if (!data) return;
    var elements = document.querySelectorAll(data.tagName || 'input');
    if (elements.length <= (data.index || 0)) return;
    var el = elements[data.index || 0];
    if (data.tagName === 'select') {
      el.value = data.value;
    } else if (el.type === 'checkbox' || el.type === 'radio') {
      el.checked = data.checked;
    }
    el.dispatchEvent(new Event('change', { bubbles: true }));
  }

  function dispatchFormEvent(data, eventType) {
    if (!data) return;
    var elements = document.querySelectorAll(data.tagName || 'form');
    if (elements.length > (data.index || 0)) {
      elements[data.index || 0].dispatchEvent(new Event(eventType, { bubbles: true, cancelable: true }));
    }
  }

  function dispatchFormReset(data) {
    if (!data) return;
    var elements = document.querySelectorAll(data.tagName || 'form');
    if (elements.length > (data.index || 0)) {
      elements[data.index || 0].reset();
    }
  }

  function showNotification(message, timeout) {
    var existing = document.querySelector('.__bs_notification');
    if (existing) existing.remove();
    var el = document.createElement('div');
    el.className = '__bs_notification';
    el.style.cssText = 'position:fixed;top:10px;right:10px;z-index:99999;padding:15px 20px;background:#333;color:#fff;border-radius:4px;font:14px/1.4 sans-serif;box-shadow:0 2px 8px rgba(0,0,0,.3);max-width:400px;transition:opacity .3s;';
    el.textContent = message;
    document.body.appendChild(el);
    setTimeout(function() {
      el.style.opacity = '0';
      setTimeout(function() { if (el.parentNode) el.remove(); }, 300);
    }, timeout);
  }

  connect();
})();
