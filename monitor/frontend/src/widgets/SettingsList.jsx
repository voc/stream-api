import React, { useEffect, useRef } from 'react';
import { useDispatch, useSelector } from 'react-redux'
import { selectStreamSettings } from '../redux/select'
import { getAllStreamSettings } from '../redux/actions'

function useDidMount() {
  const didMountRef = useRef(true);

  useEffect(() => {
    didMountRef.current = false;
  }, []);
  return didMountRef.current;
};

function SettingsItem(props) {
  const { settings } = props;
  return (<tr>
    <td data-label="Name">
      {settings.slug}
    </td>
    <td data-label="Auth">
      <input className="authKey" size="5" value={settings.secret} readOnly /><button className="secondary copyToClipboard inputAddon">Copy</button>
    </td>
    <td data-label="Notes">{settings.description}</td>
    <td style={{ textAlign: "right" }}>
      <button className="primary">Edit</button>
      <button className="secondary">Remove</button>
    </td>
  </tr>)
}

function SettingsList() {
  const didMount = useDidMount()
  const dispatch = useDispatch();
  const settings = useSelector(selectStreamSettings);

  useEffect(() => {
    console.log("effect", didMount);
    dispatch(getAllStreamSettings());
  }, [didMount])

  console.log("settings", settings);

  // return (<div>
  //   <h2>Settings</h2>
  //   <ul style={{ listStyleType: "none", paddingLeft: 0, display: "flex", flexFlow: "row wrap" }}>
  //     {Object.values(settings).map((entry) => {
  //       return <SettingsItem key={entry.slug} settings={entry} />
  //     })}
  //     {Object.values(settings).length == 0 ? <mark className="inline-block primary">No streams configured</mark> : null}
  //   </ul>
  // </div>)

  return (<div>
    <h2>Settings</h2>
    <table>
      <thead>
        <tr>
          <th>Name</th>
          <th data-label="Auth">Auth</th>
          <th>Expires</th>
          <th data-label="Notes">Notes</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {Object.values(settings).map((entry) => {
          return <SettingsItem key={entry.slug} settings={entry} />
        })}
      </tbody>
    </table>

  </div>)
}




export default SettingsList