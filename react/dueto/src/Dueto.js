import React, { Component } from 'react';
import Home from "./pages/Home.js"
import Login from "./pages/Login.js"
import Profile from "./pages/Profile.js"
import Discover from "./pages/Discover.js"
import Video from "./pages/Video.js"
import {
  BrowserRouter,
  Link,
  Route,
  Switch,
} from "react-router-dom";

class Dueto extends Component {
  render() {
    return (
      <BrowserRouter>
        <div>
          <Switch>
            <Route path="/home" component={Home} />
            <Route path="/profile" component={Profile} />
            <Route path="/login" component={Login} />
            <Route path="/discover" component={Discover} />
            <Route path="/video" component={Video} />
          </Switch>
        </div>
      </BrowserRouter>
    );
  }
}

export default Dueto;
