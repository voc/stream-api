import React from 'react'
import ReactDOM from 'react-dom'
import {Provider} from 'react-redux'
import store from './redux/store'
import {updateState, socketConnected, socketDisconnected} from './redux/actions'
import Monitor from './Monitor'
import Socket from './lib/socket'

const rootElement = document.getElementById('app')
ReactDOM.render(
  <Provider store={store}>
    <Monitor />
  </Provider>,
  rootElement
)

const socket = new Socket();
socket.on("message", (msg) => {
  if (typeof msg !== "string") {
    return
  }
  try {
    msg = JSON.parse(msg)
  } catch(err) {
    console.log("json parse", err)
  }
  console.log("msg", msg)
  store.dispatch(updateState(msg))
})
socket.on("connect", () => store.dispatch(socketConnected()));
socket.on("disconnect", () => store.dispatch(socketDisconnected()));
