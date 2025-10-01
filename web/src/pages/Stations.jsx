import React, { useState, useEffect, useCallback, useRef } from 'react';
import Sidebar from '../partials/Sidebar';
import Header from '../partials/Header';

function Stations() {
  const [stations, setStations] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [showAddModal, setShowAddModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [editingStation, setEditingStation] = useState(null);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [lastUpdate, setLastUpdate] = useState(new Date());
  
  // Use refs for form inputs to avoid re-renders completely
  const identityRef = useRef(null);
  const nameRef = useRef(null);
  const modelRef = useRef(null);
  const vendorRef = useRef(null);
  const maxOutputRef = useRef(null);

  const fetchStations = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await fetch('/api/stations');
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      setStations(data);
    } catch (error) {
      setError(error);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStations();
    
    // Set up real-time polling every 5 seconds
    const interval = setInterval(() => {
      fetchStations();
      setLastUpdate(new Date());
    }, 5000);
    
    return () => clearInterval(interval);
  }, [fetchStations]);

  const createStation = async (e) => {
    e.preventDefault();
    try {
      const payload = {
        identity: identityRef.current.value,
        name: nameRef.current.value || null,
        model: modelRef.current.value || null,
        vendor: vendorRef.current.value || null,
        max_output_kw: maxOutputRef.current.value ? parseFloat(maxOutputRef.current.value) : null
      };

      const response = await fetch('/api/stations', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || `HTTP error! status: ${response.status}`);
      }

      setShowAddModal(false);
      resetForm();
      fetchStations();
    } catch (error) {
      setError(error);
    }
  };

  const updateStation = async (e) => {
    e.preventDefault();
    if (!editingStation) return;

    try {
      const payload = {
        identity: identityRef.current.value,
        name: nameRef.current.value || null,
        model: modelRef.current.value || null,
        vendor: vendorRef.current.value || null,
        max_output_kw: maxOutputRef.current.value ? parseFloat(maxOutputRef.current.value) : null
      };

      const response = await fetch(`/api/stations/${editingStation.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || `HTTP error! status: ${response.status}`);
      }

      setShowEditModal(false);
      setEditingStation(null);
      resetForm();
      fetchStations();
    } catch (error) {
      setError(error);
    }
  };

  const deleteStation = async (id) => {
    if (!window.confirm('Are you sure you want to delete this station?')) {
      return;
    }
    try {
      const response = await fetch(`/api/stations/${id}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || `HTTP error! status: ${response.status}`);
      }

      fetchStations();
    } catch (error) {
      setError(error);
    }
  };

  const openEditModal = (station) => {
    setEditingStation(station);
    // Set ref values for editing
    if (identityRef.current) identityRef.current.value = station.identity;
    if (nameRef.current) nameRef.current.value = station.name || '';
    if (modelRef.current) modelRef.current.value = station.model || '';
    if (vendorRef.current) vendorRef.current.value = station.vendor || '';
    if (maxOutputRef.current) maxOutputRef.current.value = station.max_output_kw ? station.max_output_kw.toString() : '';
    setShowEditModal(true);
  };

  const resetForm = () => {
    if (identityRef.current) identityRef.current.value = '';
    if (nameRef.current) nameRef.current.value = '';
    if (modelRef.current) modelRef.current.value = '';
    if (vendorRef.current) vendorRef.current.value = '';
    if (maxOutputRef.current) maxOutputRef.current.value = '';
  };

  const filteredStations = stations.filter(station =>
    station.identity.toLowerCase().includes(searchTerm.toLowerCase()) ||
    (station.name && station.name.toLowerCase().includes(searchTerm.toLowerCase())) ||
    (station.model && station.model.toLowerCase().includes(searchTerm.toLowerCase())) ||
    (station.vendor && station.vendor.toLowerCase().includes(searchTerm.toLowerCase()))
  );

  const formatLastSeen = (timestamp) => {
    if (!timestamp) return 'Never';
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);
    
    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  };

  const isStationOnline = (timestamp) => {
    if (!timestamp) return false;
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);
    
    // Consider station online if last seen within 10 minutes
    return diffMins <= 10;
  };

  const formatTotalEnergy = (kwh) => {
    if (kwh === null || kwh === undefined) return '0.000 kWh';
    return `${kwh.toFixed(3)} kWh`;
  };

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Sidebar */}
      <Sidebar sidebarOpen={sidebarOpen} setSidebarOpen={setSidebarOpen} />
      
      {/* Content area */}
      <div className="relative flex flex-col flex-1 overflow-y-auto overflow-x-hidden">
        {/* Header */}
        <Header sidebarOpen={sidebarOpen} setSidebarOpen={setSidebarOpen} />
        
        {/* Main content */}
        <main className="flex-1">
          <div className="px-4 sm:px-6 lg:px-8 py-8 w-full max-w-9xl mx-auto">
      {/* Page header */}
      <div className="sm:flex sm:justify-between sm:items-center mb-8">
        {/* Left: Title with Live indicator */}
        <div className="mb-4 sm:mb-0 flex items-center gap-4">
          <h1 className="text-2xl md:text-3xl text-gray-800 dark:text-gray-100 font-bold">Stations</h1>
          {/* Live update indicator */}
          <div className="flex items-center text-sm text-gray-500 dark:text-gray-400">
            <div className="w-2 h-2 bg-green-500 rounded-full mr-2 animate-pulse"></div>
            <span>Live • Updated {formatLastSeen(lastUpdate)}</span>
          </div>
        </div>

        {/* Right: Actions */}
        <div className="grid grid-flow-col sm:auto-cols-max justify-start sm:justify-end gap-2">
          {/* Search form */}
          <div className="relative">
            <input
              type="search"
              id="search"
              className="form-input w-full pl-9"
              placeholder="Search stations..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
            <button className="absolute inset-0 right-auto group" type="submit" aria-label="Search">
              <svg className="w-4 h-4 shrink-0 fill-current text-gray-400 dark:text-gray-500 group-hover:text-gray-500 dark:group-hover:text-gray-400 ml-3 mr-2" viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                <path d="M7 14c-3.86 0-7-3.14-7-7s3.14-7 7-7 7 3.14 7 7-3.14 7-7 7ZM7 2C4.243 2 2 4.243 2 7s2.243 5 5 5 5-2.243 5-5-2.243-5-5-5Z" />
              </svg>
            </button>
          </div>
          {/* Add Station button */}
          <button className="btn bg-violet-500 hover:bg-violet-600 text-white" onClick={() => setShowAddModal(true)}>
            <svg className="w-4 h-4 fill-current opacity-50 shrink-0" viewBox="0 0 16 16">
              <path d="M15 7H9V1c0-.6-.4-1-1-1S7 .4 7 1v6H1c-.6 0-1 .4-1 1s.4 1 1 1h6v6c0 .6.4 1 1 1s1-.4 1-1V9h6c.6 0 1-.4 1-1s-.4-1-1-1Z" />
            </svg>
            <span className="hidden xs:block ml-2">Add Station</span>
          </button>
        </div>
      </div>

      {/* Table */}
      <div className="bg-white dark:bg-gray-800 shadow-lg rounded-lg border border-gray-200 dark:border-gray-700 mb-8">
        <div className="overflow-x-auto">
          <table className="table-auto w-full">
            {/* Table header */}
            <thead className="text-xs font-semibold uppercase text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-gray-900/20 border-b border-gray-200 dark:border-gray-700">
              <tr>
                <th className="px-2 py-3 whitespace-nowrap">
                  <div className="font-semibold text-left">ID</div>
                </th>
                <th className="px-2 py-3 whitespace-nowrap">
                  <div className="font-semibold text-left">OCPP Identity</div>
                </th>
                <th className="px-2 py-3 whitespace-nowrap">
                  <div className="font-semibold text-left">Model</div>
                </th>
                <th className="px-2 py-3 whitespace-nowrap">
                  <div className="font-semibold text-left">Vendor</div>
                </th>
                <th className="px-2 py-3 whitespace-nowrap">
                  <div className="font-semibold text-left">Max Output (kW)</div>
                </th>
                <th className="px-2 py-3 whitespace-nowrap">
                  <div className="font-semibold text-left">Total Energy (kWh)</div>
                </th>
                <th className="px-2 py-3 whitespace-nowrap">
                  <div className="font-semibold text-left">Status</div>
                </th>
                <th className="px-2 py-3 whitespace-nowrap">
                  <div className="font-semibold text-left">Last Heartbeat</div>
                </th>
                <th className="px-2 py-3 whitespace-nowrap">
                  <div className="font-semibold text-left">Actions</div>
                </th>
              </tr>
            </thead>
            {/* Table body */}
            <tbody className="text-sm divide-y divide-gray-200 dark:divide-gray-700">
              {loading ? (
                <tr>
                  <td colSpan="9" className="px-2 py-3 whitespace-nowrap text-center">Loading stations...</td>
                </tr>
              ) : error ? (
                <tr>
                  <td colSpan="9" className="px-2 py-3 whitespace-nowrap text-center text-red-500">Error: {error.message}</td>
                </tr>
              ) : filteredStations.length === 0 ? (
                <tr>
                  <td colSpan="9" className="px-2 py-3 whitespace-nowrap text-center">No stations found.</td>
                </tr>
              ) : (
                filteredStations.map(station => (
                  <tr key={station.id}>
                    <td className="px-2 py-3 whitespace-nowrap">
                      <div className="text-left">{station.id}</div>
                    </td>
                    <td className="px-2 py-3 whitespace-nowrap">
                      <div className="text-left font-medium text-gray-800 dark:text-gray-100">{station.identity}</div>
                    </td>
                    <td className="px-2 py-3 whitespace-nowrap">
                      <div className="text-left">{station.model || '-'}</div>
                    </td>
                    <td className="px-2 py-3 whitespace-nowrap">
                      <div className="text-left">{station.vendor || '-'}</div>
                    </td>
                    <td className="px-2 py-3 whitespace-nowrap">
                      <div className="text-left">{station.max_output_kw !== null ? `${station.max_output_kw} kW` : '-'}</div>
                    </td>
                    <td className="px-2 py-3 whitespace-nowrap">
                      <div className="text-left">{formatTotalEnergy(station.total_energy_kwh)}</div>
                    </td>
                    <td className="px-2 py-3 whitespace-nowrap">
                      <div className="text-left flex items-center">
                        {isStationOnline(station.last_seen) ? (
                          <div className="flex items-center">
                            <div className="w-2 h-2 bg-green-500 rounded-full mr-2 animate-pulse"></div>
                            <span className="text-green-600 dark:text-green-400 font-medium">Online</span>
                          </div>
                        ) : (
                          <div className="flex items-center">
                            <div className="w-2 h-2 bg-red-500 rounded-full mr-2"></div>
                            <span className="text-red-600 dark:text-red-400 font-medium">Offline</span>
                          </div>
                        )}
                      </div>
                    </td>
                    <td className="px-2 py-3 whitespace-nowrap">
                      <div className="text-left">
                        <div className="font-mono text-sm">{formatLastSeen(station.last_seen)}</div>
                      </div>
                    </td>
                    <td className="px-2 py-3 whitespace-nowrap">
                      <div className="text-left flex items-center">
                        <button
                          className="text-sm text-blue-500 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-600 mr-3"
                          onClick={() => openEditModal(station)}
                        >
                          Edit
                        </button>
                        <button
                          className="text-sm text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-600"
                          onClick={() => deleteStation(station.id)}
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Add Station Modal */}
      {showAddModal && (
        <div className="fixed inset-0 bg-gray-900 bg-opacity-75 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6 w-full max-w-lg">
            <h2 className="text-xl font-bold mb-4 text-gray-800 dark:text-gray-100">Add New Station</h2>
            <form onSubmit={createStation}>
              <div className="mb-4">
                <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="identity">
                  Identity <span className="text-red-500">*</span>
                </label>
                <input
                  ref={identityRef}
                  type="text"
                  id="identity"
                  name="identity"
                  className="form-input w-full"
                  required
                />
              </div>
              <div className="mb-4">
                <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="name">
                  Name
                </label>
                <input
                  ref={nameRef}
                  type="text"
                  id="name"
                  name="name"
                  className="form-input w-full"
                />
              </div>
              <div className="mb-4">
                <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="model">
                  Model
                </label>
                <input
                  ref={modelRef}
                  type="text"
                  id="model"
                  name="model"
                  className="form-input w-full"
                />
              </div>
              <div className="mb-4">
                <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="vendor">
                  Vendor
                </label>
                <input
                  ref={vendorRef}
                  type="text"
                  id="vendor"
                  name="vendor"
                  className="form-input w-full"
                />
              </div>
              <div className="mb-4">
                <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="max_output_kw">
                  Max Output (kW)
                </label>
                <input
                  ref={maxOutputRef}
                  type="number"
                  id="max_output_kw"
                  name="max_output_kw"
                  className="form-input w-full"
                  step="0.1"
                />
              </div>
              <div className="flex justify-end">
                <button
                  type="button"
                  className="btn bg-gray-300 dark:bg-gray-700 hover:bg-gray-400 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-100 mr-2"
                  onClick={() => {
                    setShowAddModal(false);
                    resetForm();
                  }}
                >
                  Cancel
                </button>
                <button type="submit" className="btn bg-violet-500 hover:bg-violet-600 text-white">
                  Save
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Edit Station Modal */}
      {showEditModal && (
        <div className="fixed inset-0 bg-gray-900 bg-opacity-75 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6 w-full max-w-lg">
            <h2 className="text-xl font-bold mb-4 text-gray-800 dark:text-gray-100">Edit Station</h2>
            <form onSubmit={updateStation}>
              <div className="mb-4">
                <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="edit-identity">
                  Identity <span className="text-red-500">*</span>
                </label>
                <input
                  ref={identityRef}
                  type="text"
                  id="edit-identity"
                  name="identity"
                  className="form-input w-full"
                  required
                  disabled
                />
              </div>
              <div className="mb-4">
                <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="edit-name">
                  Name
                </label>
                <input
                  ref={nameRef}
                  type="text"
                  id="edit-name"
                  name="name"
                  className="form-input w-full"
                />
              </div>
              <div className="mb-4">
                <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="edit-model">
                  Model
                </label>
                <input
                  ref={modelRef}
                  type="text"
                  id="edit-model"
                  name="model"
                  className="form-input w-full"
                />
              </div>
              <div className="mb-4">
                <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="edit-vendor">
                  Vendor
                </label>
                <input
                  ref={vendorRef}
                  type="text"
                  id="edit-vendor"
                  name="vendor"
                  className="form-input w-full"
                />
              </div>
              <div className="mb-4">
                <label className="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" htmlFor="edit-max_output_kw">
                  Max Output (kW)
                </label>
                <input
                  ref={maxOutputRef}
                  type="number"
                  id="edit-max_output_kw"
                  name="max_output_kw"
                  className="form-input w-full"
                  step="0.1"
                />
              </div>
              <div className="flex justify-end">
                <button
                  type="button"
                  className="btn bg-gray-300 dark:bg-gray-700 hover:bg-gray-400 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-100 mr-2"
                  onClick={() => {
                    setShowEditModal(false);
                    setEditingStation(null);
                    resetForm();
                  }}
                >
                  Cancel
                </button>
                <button type="submit" className="btn bg-violet-500 hover:bg-violet-600 text-white">
                  Save Changes
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
          </div>
        </main>
      </div>
    </div>
  );
}

export default Stations;