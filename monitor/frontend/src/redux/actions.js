export const UPDATE_STATE = 'UPDATE_STATE';

export const updateState = update => {
  return {
    type: UPDATE_STATE,
    ...update
  };
};


export const SOCKET_CONNECTED = 'SOCKET_CONNECTED';
export const SOCKET_DISCONNECTED = 'SOCKET_DISCONNECTED';

export const socketConnected = () => ({type: SOCKET_CONNECTED})
export const socketDisconnected = () => ({type: SOCKET_DISCONNECTED})
