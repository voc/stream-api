import React from 'react';
import {
  HashRouter as Router,
  Switch,
  Route,
} from "react-router-dom";
import Header from './widgets/Header';
import SpinnerOverlay from './widgets/SpinnerOverlay'
import Monitor from './Monitor';
import Settings from './Settings';

function App() {

  return <Router><>
    <Header></Header>

    <main className="container">
      <Switch>
        <Route path="/settings">
          <Settings />
        </Route>
        <Route path="/">
          <Monitor />
        </Route>
      </Switch>
      <SpinnerOverlay />
    </main>

    <footer>
      <p className="app-footer">
        Made by <a
          target="_blank"
          rel="noopener noreferrer"
          href="https://c3voc.de"
        >c3voc</a
        >.
      </p>
    </footer>
  </></Router>;
}

export default App
