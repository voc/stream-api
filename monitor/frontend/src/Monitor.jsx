import React from 'react';
import StreamList from './widgets/StreamList';
import TranscoderList from './widgets/TranscoderList';
import FanoutList from './widgets/FanoutList';

function Monitor() {
    return <>
      <StreamList />
      <TranscoderList />
      <FanoutList />
    </>;
  }
  
  export default Monitor