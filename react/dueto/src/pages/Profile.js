import React, { Component } from 'react'
import { Paper, Tabs, Tab, IconButton, TextField } from 'material-ui'
import AccountCircle from 'material-ui-icons/AccountCircle'
import {getProfileData} from '../utils/fetchData.js'

class Profile extends Component {
  constructor() {
    super()
  }

  componentDidMount() {
    console.log("did mount")
    getProfileData("joeshmow")
      .then(data => {
        console.log(data)
      })
      .catch(error => {})
  }

  render() {
    return (
      <div>
        <div>
          <label>profile page</label>
        </div>
      </div>
    )
  }
}

export default Profile;
