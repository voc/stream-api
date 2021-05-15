import EventEmitter from 'eventemitter3'

export default class Socket extends EventEmitter {
  constructor() {
    super()
    this.reconnect()
  }

  send() {
    this.socket.apply(this.socket, arguments);
  }

  reconnect() {
    const proto = location.protocol == "http:" ? "ws:" : "wss:"
    this.socket = new WebSocket(`${proto}//${location.host}/ws`);
    this.socket.onopen = this.handleOpen.bind(this)
    this.socket.onmessage = this.handleMessage.bind(this)
    this.socket.onclose = this.handleClose.bind(this)
    this.socket.onerror = this.handleError.bind(this)
  }

  handleOpen() {
    this.emit("connect");
  }

  handleMessage(event) {
    this.emit("message", event.data)
  }

  handleClose(event) {
    if (event.wasClean) {
      // console.log(`[close] Connection closed cleanly, code=${event.code} reason=${event.reason}`);
    } else {
      // console.log('[close] Connection died');
    }
    this.emit("disconnect");
    console.log("connection ended, reconnecting")
    setTimeout(this.reconnect.bind(this), 1000);
  }

  handleError(error) {
    // console.log(`[error] ${error.message}`);
  }
}



