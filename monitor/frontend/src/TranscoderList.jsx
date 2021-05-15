import React from 'react';
import {useSelector} from 'react-redux'
import {selectTranscoders} from './redux/select'

function TranscoderItem(props) {
  const {transcoder} = props;
  return <li className="card fluid">
    <div className="section">
      <h4>{transcoder.name}</h4>
    </div>
    <div className="section">
      <p>Capacity: {transcoder.capacity}</p>
      <p>Streams: {transcoder.streams.length}</p>
    </div>
  </li>
}

function TranscoderList() {
  const transcoders = useSelector(selectTranscoders);

  return (<div>
    <h2>Transcoders</h2>
    <ul style={{listStyleType: "none", paddingLeft: 0, display: "flex", flexFlow: "row wrap"}}>
      {Object.values(transcoders).map((transcoder) => {
        return <TranscoderItem key={transcoder.name} transcoder={transcoder}/>
      })}
      {Object.values(transcoders).length == 0 ? <mark className="inline-block secondary">No transcoders registered</mark> : null}
    </ul>
  </div>)
}

export default TranscoderList
