import React from 'react';
import SocketState from './SocketState';
import StreamList from './StreamList';
import TranscoderList from './TranscoderList';
import FanoutList from './FanoutList';

function Monitor() {
  return <main className="container">
    <h1>Stream-API Monitor<SocketState/></h1>
    <StreamList/>
    <TranscoderList/>
    <FanoutList/>

    <p className="app-footer">
    Made by <a
      target="_blank"
      rel="noopener noreferrer"
      href="https://c3voc.de"
      >c3voc</a
    >.
  </p>
  </main>
}

export default Monitor
