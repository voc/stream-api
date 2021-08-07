import React from 'react';
import { useSelector } from 'react-redux';
import SettingsForm from './widgets/SettingsForm';
import SettingsList from './widgets/SettingsList';

function Settings() {
  return <>
    <SettingsList/>

    <SettingsForm />
  </>;
}

export default Settings