import React from 'react';
import {useSelector} from 'react-redux'
import {selectFanouts} from '../redux/select'

function FanoutItem(props) {
  const {fanout} = props;
  return <li className="card fluid">
    <div className="section">
      <h4>{fanout.name}</h4>
    </div>
    <div className="section">
      <p>Sink: {fanout.sink}</p>
    </div>
  </li>
}

function FanoutList() {
  const fanouts = useSelector(selectFanouts);

  return (<div>
    <h2>Fanouts</h2>
    <ul style={{listStyleType: "none", paddingLeft: 0, display: "flex", flexFlow: "row wrap"}}>
      {Object.values(fanouts).map((fanout) => {
        return <FanoutItem key={fanout.name} fanout={fanout}/>
      })}
      {Object.values(fanouts).length == 0 ? <mark className="inline-block secondary">No fanouts registered</mark> : null}
    </ul>
  </div>)
}

export default FanoutList
