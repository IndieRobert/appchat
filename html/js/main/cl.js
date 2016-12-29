/*
 * client 
 */

cl = (function() {
  var _cl                 = {}
  var self                = _cl;
  var _serverPort         = "3000"
  var _serverEndPoint     = "http://127.0.0.1:" + _serverPort
  // FIXME
  // There must be better ways of doing this like getting the code from server or from a file. ReactJS?
  // Anyway, the idea is we want one html page and switch html nodes.
  var _claimUsernameUI   =  "\
    <div id='claimUsername'> \
      <div class='hero-unit'> \
        <form> \
          <div> <input id='username' pattern='[A-Za-z1-9\-_]+' type='text' name='name' placeholder='username'> </div> \
          <div> <a class='chat btn btn-large btn-primary' href='javascript:cl.chatNow()'>Chat now!</a></div> \
          <div id='flash'></div> \
        </form> \
      </div> \
    </div>"
  // FIXME
  // There must be better ways of doing this like getting the code from server or from a file. ReactJS?
  // Anyway, the idea is we want one html page and switch html nodes.
  var _chatRoomUI         =  "\
    <div id='chatRoomUI'> \
      <div class='hero-unit'> \
        <form> \
          <div> <input id='messagebox' type='text' name='name' placeholder='Type your message...'> </div> \
          <div> <a class='chat btn btn-large btn-primary' href='javascript:cl.sendMessage()'>Send</a></div> \
          <div id='flash'></div> \
          <textarea id='chatbox' rows='20' cols='1050' readonly/>\
        </form> \
      </div> \
    </div>"

   _cl.initClaimUsernameUI = function() {
    $("#ac").append(_claimUsernameUI)
    $("#username").focus();
  }

  _cl.initChatRoomUI = function() {
    $("#messagebar").focus();
  }

  _cl.swapForUI = function(ui) {
    // SUPER FIXME
    // Remove all children of ac instead
    $("#claimUsername").remove()
    $("#ac").append(ui)
  }

  _cl.getConnection = function(successCB) {
    self.connection = new WebSocket('ws://127.0.0.1:' + _serverPort + '/ws/')
    // FIXME
    // Shouldnt this be before we establish connection with server?
    self.connection.onopen = function () {
      console.log("onopen")
      successCB()
    };
    self.connection.onclose = function () {
      console.log("onclose")
    };
    self.connection.onerror = function (error) {
      console.log('WebSocket Error ' + error);
    };
    self.connection.onmessage = function (e) {
      console.log('Got message from server: ' + e.data);
      var cmd = JSON.parse(JSON.parse(e.data)) // FIXME: 2x parse
      switch(cmd.type) {
        case "claimUsername":
          if(cmd.success) {
            console.log("Entering chat room.")
            self.swapForUI(_chatRoomUI)
            self.initChatRoomUI()
            self.connection.send(JSON.stringify({
              "type": "getHistory"
            }));
          } else {
            $("#flash").html("<span class='warn'>User name already taken</span>");
          }
          break
        case "getHistory":
          console.log(cmd.history)
          cmd.history.forEach(function(historyMessage) {
            self.appendMessage(historyMessage.author, historyMessage.message)
          })
          $("#messagebar").focus();
          break
        case "sendMessage":
          cmd.messages.forEach(function(message) {
            self.appendMessage(message.author, message.message)
          })
          $("#messagebar").focus();
          break
      }
    };
  }

  _cl.claimUsername = function(username, successCB) {
    console.log(JSON.stringify({
      "type": "claimUsername",
      "username": username
    }))
    self.connection.send(JSON.stringify({
      "type": "claimUsername",
      "username": username
    }));
  }

  _cl.chatNow = function() {
    self.getConnection(function() {
      var username = $("#username").val()
      if(username != "") {
        self.claimUsername(username)
        $("#flash").html("");
      } else {
        $("#flash").html("<span class='warn'>User name is empty</span>");
      }
    })
  }

  _cl.sendMessage = function() {
    var message = $("#messagebox").val()
    if(message != "") {
      self.connection.send(JSON.stringify({
        "type": "sendMessage",
        "message": message
      }));
      $("#messagebox").val("")
      $("#flash").html("");
    } else {
      $("#flash").html("<span class='warn'>Message is empty</span>");
    }
  }

  _cl.appendMessage = function(author, message) {
    $('#chatbox').append(author + ": " + message + "\n");
  }

  $("#flash").html("<span class='success'>This is a flash</span>");

  return _cl
})()