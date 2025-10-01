import React, { useState, useEffect } from 'react';

import Sidebar from '../partials/Sidebar';
import Header from '../partials/Header';
import Banner from '../partials/Banner';

function OCPPSettings() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState(null);
  const [serverStatus, setServerStatus] = useState({
    ocppServerRunning: false,
    httpAddr: ':8080',
    ocppEndpoint: '/ocpp16'
  });
  const [serverActionLoading, setServerActionLoading] = useState(false);
  const [settings, setSettings] = useState({
    heartbeatInterval: '30'
  });
  const [networkInfo, setNetworkInfo] = useState({
    hotspot: {
      ip_address: '192.168.137.1',
      subnet_mask: '255.255.255.0',
      is_reachable: true
    }
  });

  // Fetch settings and server status on component mount
  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        setError(null);
        
        // Fetch settings
        const settingsResponse = await fetch('/api/settings');
        if (!settingsResponse.ok) {
          throw new Error(`HTTP error! status: ${settingsResponse.status}`);
        }
        const settingsData = await settingsResponse.json();
        setSettings(settingsData);

        // Fetch server status
        const statusResponse = await fetch('/api/settings/status');
        if (!statusResponse.ok) {
          throw new Error(`HTTP error! status: ${statusResponse.status}`);
        }
        const statusData = await statusResponse.json();
        setServerStatus(statusData);

        // Fetch hotspot information (lightweight, no network checking)
        const hotspotResponse = await fetch('/api/network/hotspot');
        if (hotspotResponse.ok) {
          const hotspotData = await hotspotResponse.json();
          setNetworkInfo({
            hotspot: hotspotData
          });
        }
      } catch (error) {
        setError(error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  const handleServerAction = async (action) => {
    try {
      setServerActionLoading(true);
      setError(null);
      
      const response = await fetch(`/api/settings/server/${action}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || `HTTP error! status: ${response.status}`);
      }

      const result = await response.json();
      alert(result.message);
      
      // Refresh server status
      const statusResponse = await fetch('/api/settings/status');
      if (statusResponse.ok) {
        const statusData = await statusResponse.json();
        setServerStatus(statusData);
      }
    } catch (error) {
      setError(error);
    } finally {
      setServerActionLoading(false);
    }
  };

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setSettings(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const handleSave = async (e) => {
    e.preventDefault();
    try {
      setSaving(true);
      setError(null);
      
      const response = await fetch('/api/settings', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(settings),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || `HTTP error! status: ${response.status}`);
      }

      alert('Settings saved successfully!');
    } catch (error) {
      setError(error);
    } finally {
      setSaving(false);
    }
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

      {/* Error Message */}
      {error && (
        <div className="mb-6 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800 dark:text-red-200">Error</h3>
              <div className="mt-2 text-sm text-red-700 dark:text-red-300">
                {error.message}
              </div>
            </div>
          </div>
        </div>
      )}


      {/* Connection Status */}
      <div className="bg-white dark:bg-gray-800 shadow-lg rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <h3 className="text-lg font-semibold text-gray-800 dark:text-gray-100 mb-4">Connection Status</h3>
        <div className="flex items-center space-x-3">
          <div className={`w-3 h-3 rounded-full ${serverStatus.ocppServerRunning ? 'bg-green-500' : 'bg-red-500'}`}></div>
          <span className="text-gray-700 dark:text-gray-300">
            OCPP Server: {serverStatus.ocppServerRunning ? 'Running' : 'Stopped'}
          </span>
        </div>
        <div className="mt-2 space-y-1">
          <p className="text-sm text-gray-500 dark:text-gray-400">
            <strong>HTTP Server:</strong> {serverStatus.httpAddr}
          </p>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            <strong>OCPP Endpoint:</strong> {serverStatus.ocppEndpoint}
          </p>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            <strong>Charger WebSocket URL:</strong>
          </p>
          <div className="bg-gray-100 dark:bg-gray-700 rounded p-2 mt-1">
            <code className="text-sm font-mono text-blue-600 dark:text-blue-400">
              ws://{networkInfo.hotspot.ip_address}:8080/ocpp16/[CHARGER_ID]
            </code>
          </div>
        </div>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-3">
          {serverStatus.ocppServerRunning 
            ? 'The OCPP 1.6J server is running and ready to accept station connections.'
            : 'The OCPP 1.6J server is currently stopped. Restart the application to enable it.'
          }
        </p>
        
        {/* Server Control Buttons */}
        <div className="mt-4 flex space-x-3">
          <button
            onClick={() => handleServerAction('start')}
            disabled={serverActionLoading || serverStatus.ocppServerRunning}
            className={`px-4 py-2 rounded-md text-sm font-medium ${
              serverActionLoading || serverStatus.ocppServerRunning
                ? 'bg-gray-300 dark:bg-gray-600 text-gray-500 dark:text-gray-400 cursor-not-allowed'
                : 'bg-green-600 hover:bg-green-700 text-white'
            }`}
          >
            {serverActionLoading ? 'Processing...' : 'Start Server'}
          </button>
          
          <button
            onClick={() => handleServerAction('stop')}
            disabled={serverActionLoading || !serverStatus.ocppServerRunning}
            className={`px-4 py-2 rounded-md text-sm font-medium ${
              serverActionLoading || !serverStatus.ocppServerRunning
                ? 'bg-gray-300 dark:bg-gray-600 text-gray-500 dark:text-gray-400 cursor-not-allowed'
                : 'bg-red-600 hover:bg-red-700 text-white'
            }`}
          >
            {serverActionLoading ? 'Processing...' : 'Stop Server'}
          </button>
        </div>
      </div>

      {/* Settings Form */}
      <div className="mt-6 bg-white dark:bg-gray-800 shadow-lg rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        {loading ? (
          <div className="text-center py-8">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-violet-500"></div>
            <p className="mt-2 text-gray-600 dark:text-gray-400">Loading settings...</p>
          </div>
        ) : (
          <form onSubmit={handleSave}>
            <div className="space-y-6">
              <h3 className="text-lg font-semibold text-gray-800 dark:text-gray-100 mb-4">OCPP Configuration</h3>
              
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
                  placeholder="30"
                  min="30"
                  max="3600"
                  required
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">How often stations send heartbeat messages</p>
              </div>

            </div>

          {/* Action Buttons */}
          <div className="flex justify-end space-x-3 mt-8 pt-6 border-t border-gray-200 dark:border-gray-700">
            <button
              type="submit"
              disabled={saving}
              className="btn bg-violet-500 hover:bg-violet-600 text-white disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {saving ? 'Saving...' : 'Save Settings'}
            </button>
          </div>
        </form>
        )}
      </div>

          </div>
        </main>

        <Banner />
      </div>
    </div>
  );
}

export default OCPPSettings;
