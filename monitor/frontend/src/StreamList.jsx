import React from 'react';
import {useSelector} from 'react-redux'
import {selectStreams, selectTranscoders, selectStreamTranscoders} from './redux/select'
import StreamItem from './StreamItem'

function StreamList() {
  const streams = useSelector(selectStreams),
    transcoders = useSelector(selectTranscoders),
    streamTranscoders = useSelector(selectStreamTranscoders);

  return (<div>
    <h2>Streams</h2>
    <ul style={{listStyleType: "none", paddingLeft: 0, display: "flex", flexFlow: "row wrap"}}>
      {Object.entries(streams).map(([slug, value]) => {
        const transcoderName = streamTranscoders[slug];
        const transcoder = transcoders[transcoderName];
        return <StreamItem key={slug} stream={value} transcoder={transcoder}/>
      })}
      {Object.values(streams).length == 0 ? <mark className="inline-block">No streams detected</mark> : null}
    </ul>
  </div>)
}

export default StreamList
