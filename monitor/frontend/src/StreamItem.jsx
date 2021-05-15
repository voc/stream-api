import React from 'react';

function StreamItem(props) {
  const {stream, transcoder} = props;
  return <li className="card fluid">
    <div className="section">
      <h4>{stream.slug}</h4>
    </div>
    <div className="section">
      <p>Format: {stream.format}</p>
      <p>Source: {stream.source}</p>
    </div>
    {transcoder ?
    <div className="section">
      <p>Transcoder: {transcoder.name}</p>
    </div>
    : null}
  </li>
}

export default StreamItem
