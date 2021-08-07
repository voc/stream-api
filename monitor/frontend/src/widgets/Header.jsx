import React from "react";
import { Link } from "react-router-dom";
import SocketState from './SocketState';

function Header() {
  return <header>
    <Link to="/" className="button">Monitor</Link>
    <Link to="/settings" className="button">Settings</Link>
    <div className="button"><SocketState className="button" /></div>
  </header>
}

export default Header;