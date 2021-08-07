import { createStore, applyMiddleware, compose } from 'redux';
import thunk from 'redux-thunk';
import { reducer } from './reducer.js';

const extension = window.__REDUX_DEVTOOLS_EXTENSION__ && window.__REDUX_DEVTOOLS_EXTENSION__()
const enhancers = compose(applyMiddleware(thunk), extension)
const store = createStore(reducer, enhancers);

export default store
