import React from 'react';
import {useSelector} from 'react-redux'
import {selectSocketConnected} from './redux/select'

function SocketState() {
  const connected = useSelector(selectSocketConnected);
  return <mark className={"tag " + (connected ? "tertiary" : "secondary")} style={{fontSize: "14px", margin: "0 15px", verticalAlign: "middle"}}>
    {connected ? "connected" : "disconnected"}
  </mark>
}

export default SocketState
