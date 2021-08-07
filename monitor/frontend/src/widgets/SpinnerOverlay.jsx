import React from 'react';
import {useSelector} from 'react-redux'
import {selectSocketConnected} from '../redux/select'

function SpinnerOverlay() {
  const connected = useSelector(selectSocketConnected);
  if (connected)
    return null;

  return <div className="spinnerOverlay">
      <div className="spinner primary"></div>
  </div>;
}

export default SpinnerOverlay
