import React, { useState, useEffect } from 'react';
import Sidebar from '../partials/Sidebar';
import Header from '../partials/Header';

function Logs() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [logsConfig, setLogsConfig] = useState({
    enabled: false,
    directory: '',
    frequency: 'hours',
    frequency_value: 1,
    lastExport: null
  });

  // Fetch logs configuration on component mount
  useEffect(() => {
    const fetchConfig = async () => {
      try {
        const response = await fetch('/api/logs/config');
        if (response.ok) {
          const config = await response.json();
          setLogsConfig(config);
        }
      } catch (error) {
        console.error('Failed to fetch logs config:', error);
      }
    };

    fetchConfig();
  }, []);

  const handleInputChange = (e) => {
    const { name, value, type, checked } = e.target;
    setLogsConfig(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : (name === 'frequency_value' ? parseInt(value) || 1 : value)
    }));
  };

  const handleSaveConfig = async (e) => {
    e.preventDefault();
    try {
      const response = await fetch('/api/logs/config', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(logsConfig),
      });

      if (response.ok) {
        alert('Configuration saved successfully!');
      } else {
        alert('Failed to save configuration');
      }
    } catch (error) {
      console.error('Error saving config:', error);
      alert('Error saving configuration');
    }
  };

  const handleDownloadCSV = async () => {
    try {
      const response = await fetch('/api/logs/download', { method: 'POST' });
      if (response.ok) {
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'stations_log.csv';
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
        alert('CSV downloaded successfully!');
      } else {
        alert('Failed to download CSV');
      }
    } catch (error) {
      console.error('Download failed:', error);
      alert('Download failed');
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
              <h1 className="text-2xl md:text-3xl text-gray-800 dark:text-gray-100 font-bold">Logs</h1>
              <p className="text-gray-600 dark:text-gray-400 mt-2">Configure CSV log exports and download station data</p>
            </div>

            {/* Configuration Form */}
            <div className="bg-white dark:bg-gray-800 shadow-lg rounded-lg">
              <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
                <h2 className="text-lg font-semibold text-gray-800 dark:text-gray-100">Logs Configuration</h2>
              </div>
              
              <form onSubmit={handleSaveConfig} className="p-6 space-y-6">
                {/* Enable Logs */}
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    id="enabled"
                    name="enabled"
                    checked={logsConfig.enabled}
                    onChange={handleInputChange}
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label htmlFor="enabled" className="ml-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    Enable automatic CSV log exports
                  </label>
                </div>

                {/* Directory */}
                <div>
                  <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="directory">
                    Save Directory
                  </label>
                  <input
                    type="text"
                    id="directory"
                    name="directory"
                    value={logsConfig.directory}
                    onChange={handleInputChange}
                    className="form-input w-full"
                    placeholder="C:\logs"
                    disabled={!logsConfig.enabled}
                  />
                  <div className="mt-1">
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                      <strong>Type the full directory path (e.g., C:\logs).</strong> The directory will be created automatically if it doesn't exist.
                    </p>
                  </div>
                </div>

                {/* Frequency */}
                <div>
                  <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="frequency">
                    Export Frequency
                  </label>
                  <div className="flex gap-2">
                    <select
                      id="frequency"
                      name="frequency"
                      value={logsConfig.frequency}
                      onChange={handleInputChange}
                      className="form-select flex-1"
                    >
                      <option value="seconds">Seconds</option>
                      <option value="minutes">Minutes</option>
                      <option value="hours">Hours</option>
                    </select>
                    <input
                      type="number"
                      id="frequency_value"
                      name="frequency_value"
                      value={logsConfig.frequency_value}
                      onChange={handleInputChange}
                      className="form-input w-20"
                      min="1"
                      max="999"
                    />
                  </div>
                </div>

                {/* Save Button */}
                <div className="flex justify-end">
                  <button
                    type="submit"
                    className="btn bg-blue-500 hover:bg-blue-600 text-white px-6 py-2 rounded-lg"
                  >
                    Save Configuration
                  </button>
                </div>
              </form>
            </div>

            {/* Download Section */}
            <div className="mt-8 bg-white dark:bg-gray-800 shadow-lg rounded-lg">
              <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
                <h2 className="text-lg font-semibold text-gray-800 dark:text-gray-100">Download CSV</h2>
              </div>
              <div className="p-6">
                <p className="text-gray-600 dark:text-gray-400 mb-4">
                  Download a CSV file with current station data.
                </p>
                <button
                  onClick={handleDownloadCSV}
                  className="btn bg-green-500 hover:bg-green-600 text-white px-6 py-2 rounded-lg"
                >
                  Download CSV Now
                </button>
              </div>
            </div>
          </div>
        </main>
      </div>
    </div>
  );
}

export default Logs;