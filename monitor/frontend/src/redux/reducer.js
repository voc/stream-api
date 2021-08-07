import { SOCKET_CONNECTED, SOCKET_DISCONNECTED, UPDATE_STATE, UPDATED_STREAM_SETTINGS } from "./actions"

const INITIAL_STATE = {
  streams: {},
  transcoders: {},
  streamTranscoders: {},
  streamSettings: {},
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
    case UPDATED_STREAM_SETTINGS:
      const settings = {};
      action.settings.forEach((item) => {
        settings[item.slug] = item;
      })
      return {
        ...state,
        streamSettings: settings,
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
