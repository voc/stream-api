import {SOCKET_CONNECTED, SOCKET_DISCONNECTED, UPDATE_STATE} from "./actions"

const INITIAL_STATE = {
  streams: {},
  transcoders: {},
  fanouts: {},
  socketConnected: false,
};

export const reducer = (state = INITIAL_STATE, action) => {
  switch (action.type) {
    case UPDATE_STATE:
      delete action.type
      return {
        ...state,
        ...action,
      }
    case SOCKET_CONNECTED:
      return {
        ...state,
        socketConnected: true
      }
    case SOCKET_DISCONNECTED:
      return {
        ...state,
        socketConnected: false
      }
    default:
      return state;
  }
};
