package goui

const javascript = `
	(function(goui) {
		var messageHandlers = {};	
		var messageQueue = [];
		var windowId = 0;
		var jsReady = false;
		goui.SetMessageHandler = function(message, handler) {
			messageHandlers[message] = handler;
		}		
		
		goui.Init = function() {
			jsReady = true;
			internalInit();
		}
		
		goui.SendMessage = function(type, params, options) {
			var messageSpec = {
				type: type,
				params: params,
				options: options
			};
			if ( windowId > 0 )
				internalSendMessage(messageSpec);
			else
				messageQueue.push(messageSpec);
		}
		
		goui.CloseWindow = function() {
			goui.SendMessage('goui.closeWindow')
		}
		
		function internalSendMessage(messageSpec) {
			var params = messageSpec.params || {};
			var options = messageSpec.options || {};
			var type = messageSpec.type;
				
			var xhr = new XMLHttpRequest();

			if (options.timeout) {
				xhr.timeout = options.timeout;
				if (options.complete)
					xhr.ontimeout = options.complete;
			}
						
			xhr.onreadystatechange = function() {
				if (xhr.readyState == XMLHttpRequest.DONE) {
					if(xhr.status == 200) {
						if (options.success) {
							var object = JSON.parse(xhr.responseText);
							options.success(object);
						}
					}
					
					if (options.complete)
						options.complete();
				}
			};
			alert( 'xhr' );
			
			params.windowId = windowId;
			data = JSON.stringify({
				Type: type,
				Params: params
			});
			
			xhr.open('POST', '/callback', true);
			xhr.setRequestHeader('Content-type', 'application/json; charset=utf-8');
			xhr.send(data);
		}
		
		function internalInit() {
			if (jsReady && windowId > 0) {
				asyncLongPoll();

				for (var i = 0; i < messageQueue.length; i++)
					goui.SendMessage(messageQueue[i].type, messageQueue[i].params, messageQueue[i].options);					
				messageQueue = [];
			}
		}		
		
		function asyncLongPoll() {
			setTimeout(longPoll, 0);
		}
		
		function longPoll() {
			goui.SendMessage('goui.longPoll', {}, {
				timeout: 300000000,
				complete: asyncLongPoll,
				success: function(message) {
					if (messageHandlers[message.Type])
						messageHandlers[message.Type](message.Params);
					else
						log("Warning: unknown message received: " + data.message);
				}
			});
		}
		
		function log(message) {
			if (console && console.log)
				console.log(message);
		}
		
		goui._setWindowId = function(id) {
			windowId = id;
			internalInit();
		}
	})(window.goui = window.goui || {});
`