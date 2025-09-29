import React, { useState } from 'react';

import Sidebar from '../partials/Sidebar';
import Header from '../partials/Header';
import Banner from '../partials/Banner';

function OCPPSettings() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [settings, setSettings] = useState({
    serverPort: '8081',
    heartbeatInterval: '300',
    connectionTimeout: '30',
    maxConnections: '100',
    enableLogging: true,
    logLevel: 'info'
  });

  const handleInputChange = (e) => {
    const { name, value, type, checked } = e.target;
    setSettings(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value
    }));
  };

  const handleSave = (e) => {
    e.preventDefault();
    // TODO: Implement save to backend
    console.log('Saving OCPP settings:', settings);
    // For now, just show a success message
    alert('Settings saved successfully!');
  };

  const handleTestConnection = () => {
    // TODO: Implement connection test
    console.log('Testing OCPP connection...');
    alert('Connection test initiated!');
  };

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Sidebar */}
      <Sidebar sidebarOpen={sidebarOpen} setSidebarOpen={setSidebarOpen} />

      {/* Content area */}
      <div className="relative flex flex-col flex-1 overflow-y-auto overflow-x-hidden">
        {/* Site header */}
        <Header sidebarOpen={sidebarOpen} setSidebarOpen={setSidebarOpen} />

        <main className="grow">
          <div className="px-4 sm:px-6 lg:px-8 py-8 w-full max-w-4xl mx-auto">
      {/* Page header */}
      <div className="mb-8">
        <h1 className="text-2xl md:text-3xl text-gray-800 dark:text-gray-100 font-bold">OCPP Settings</h1>
        <p className="text-gray-600 dark:text-gray-400 mt-2">Configure OCPP 1.6J server connection settings</p>
      </div>

      {/* Settings Form */}
      <div className="bg-white dark:bg-gray-800 shadow-lg rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <form onSubmit={handleSave}>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Server Configuration */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-gray-800 dark:text-gray-100 mb-4">Server Configuration</h3>
              
              <div>
                <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="serverPort">
                  Server Port
                </label>
                <input
                  type="number"
                  id="serverPort"
                  name="serverPort"
                  value={settings.serverPort}
                  onChange={handleInputChange}
                  className="form-input w-full"
                  placeholder="8081"
                  min="1024"
                  max="65535"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Port for OCPP WebSocket connections</p>
              </div>

              <div>
                <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="heartbeatInterval">
                  Heartbeat Interval (seconds)
                </label>
                <input
                  type="number"
                  id="heartbeatInterval"
                  name="heartbeatInterval"
                  value={settings.heartbeatInterval}
                  onChange={handleInputChange}
                  className="form-input w-full"
                  placeholder="300"
                  min="30"
                  max="3600"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">How often stations send heartbeat messages</p>
              </div>

              <div>
                <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="connectionTimeout">
                  Connection Timeout (seconds)
                </label>
                <input
                  type="number"
                  id="connectionTimeout"
                  name="connectionTimeout"
                  value={settings.connectionTimeout}
                  onChange={handleInputChange}
                  className="form-input w-full"
                  placeholder="30"
                  min="5"
                  max="300"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Timeout for establishing connections</p>
              </div>
            </div>

            {/* Advanced Settings */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-gray-800 dark:text-gray-100 mb-4">Advanced Settings</h3>
              
              <div>
                <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="maxConnections">
                  Max Connections
                </label>
                <input
                  type="number"
                  id="maxConnections"
                  name="maxConnections"
                  value={settings.maxConnections}
                  onChange={handleInputChange}
                  className="form-input w-full"
                  placeholder="100"
                  min="1"
                  max="1000"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Maximum concurrent station connections</p>
              </div>

              <div>
                <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="logLevel">
                  Log Level
                </label>
                <select
                  id="logLevel"
                  name="logLevel"
                  value={settings.logLevel}
                  onChange={handleInputChange}
                  className="form-select w-full"
                >
                  <option value="debug">Debug</option>
                  <option value="info">Info</option>
                  <option value="warn">Warning</option>
                  <option value="error">Error</option>
                </select>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Level of detail in OCPP logs</p>
              </div>

              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="enableLogging"
                  name="enableLogging"
                  checked={settings.enableLogging}
                  onChange={handleInputChange}
                  className="form-checkbox"
                />
                <label htmlFor="enableLogging" className="ml-2 text-gray-700 dark:text-gray-300 text-sm">
                  Enable OCPP Logging
                </label>
              </div>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex justify-end space-x-3 mt-8 pt-6 border-t border-gray-200 dark:border-gray-700">
            <button
              type="button"
              onClick={handleTestConnection}
              className="btn bg-blue-500 hover:bg-blue-600 text-white"
            >
              Test Connection
            </button>
            <button
              type="submit"
              className="btn bg-violet-500 hover:bg-violet-600 text-white"
            >
              Save Settings
            </button>
          </div>
        </form>
      </div>

      {/* Connection Status */}
      <div className="mt-6 bg-white dark:bg-gray-800 shadow-lg rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <h3 className="text-lg font-semibold text-gray-800 dark:text-gray-100 mb-4">Connection Status</h3>
        <div className="flex items-center space-x-3">
          <div className="w-3 h-3 bg-red-500 rounded-full"></div>
          <span className="text-gray-700 dark:text-gray-300">OCPP Server: Disabled</span>
        </div>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">
          The OCPP 1.6J server is currently disabled. Enable it in the settings above to start accepting station connections.
        </p>
      </div>
          </div>
        </main>

        <Banner />
      </div>
    </div>
  );
}

export default OCPPSettings;
