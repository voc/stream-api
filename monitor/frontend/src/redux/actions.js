import { get, post } from "../lib/ajax";

export const UPDATE_STATE = 'UPDATE_STATE';

export const updateState = update => {
  return {
    type: UPDATE_STATE,
    ...update
  };
};


export const SOCKET_CONNECTED = 'SOCKET_CONNECTED';
export const SOCKET_DISCONNECTED = 'SOCKET_DISCONNECTED';

export const socketConnected = () => ({ type: SOCKET_CONNECTED })
export const socketDisconnected = () => ({ type: SOCKET_DISCONNECTED })

export const SET_STREAM_SETTINGS = 'SET_STREAM_SETTINGS'
export const UPDATED_STREAM_SETTINGS = 'UPDATED_STREAM_SETTINGS'
export const FETCH_ERROR = 'FETCH_ERROR'

export const setStreamSettings = (settings) => {
  return function (dispatch) {
    post(`/stream/${settings.slug}/settings`, settings)
      .then((res) => {
        dispatch({ type: SET_STREAM_SETTINGS, settings: res })
      })
      .catch((err) => {
        dispatch({ type: FETCH_ERROR, error: err })
      })
  }
}

export const getAllStreamSettings = () => {
  return function (dispatch) {
    get(`/stream/settings`)
      .then((res) => {
        dispatch({ type: UPDATED_STREAM_SETTINGS, settings: res })
      })
      .catch((err) => {
        dispatch({ type: FETCH_ERROR, error: err })
      })
  }
}


